package bot

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gempir/go-twitch-irc/v4"
	"github.com/mayor-wren/porygon/database"
)

type LogEntry struct {
	Timestamp time.Time
	Username  string
	Command   string
}

type Config struct {
	BotUsername string
	Channel     string
	AccessToken string
}

type Bot struct {
	config *Config
	db     *sql.DB
	client *twitch.Client

	mutex           sync.RWMutex
	commands        map[string]*database.Command
	aliases         map[string]string
	keywords        []*database.Command
	keywordRegexps  []*regexp.Regexp
	lastFired       map[string]time.Time
	restartCh       chan struct{}
	startCh         chan struct{}
	connCancel      context.CancelFunc
	onAuthFailed    func()
	activityLog     []LogEntry

	connected bool
	stopped   bool
}

func New(config *Config, db *sql.DB) *Bot {
	return &Bot{
		config:    config,
		db:        db,
		commands:  make(map[string]*database.Command),
		aliases:   make(map[string]string),
		lastFired: make(map[string]time.Time),
		restartCh: make(chan struct{}, 1),
		startCh:   make(chan struct{}, 1),
		stopped:   true,
	}
}

func (bot *Bot) IsConnected() bool {
	bot.mutex.RLock()
	defer bot.mutex.RUnlock()
	return bot.connected
}

func (bot *Bot) IsStopped() bool {
	bot.mutex.RLock()
	defer bot.mutex.RUnlock()
	return bot.stopped
}

func (bot *Bot) StartBot() {
	bot.mutex.Lock()
	bot.stopped = false
	bot.mutex.Unlock()

	select {
	case bot.startCh <- struct{}{}:
	default:
	}
}

func (bot *Bot) StopBot() {
	bot.mutex.Lock()
	bot.stopped = true
	cancel := bot.connCancel
	bot.mutex.Unlock()

	if cancel != nil {
		cancel()
	}
}

func (bot *Bot) LogCommand(username, command string) {
	bot.mutex.Lock()
	defer bot.mutex.Unlock()
	bot.activityLog = append(bot.activityLog, LogEntry{
		Timestamp: time.Now(),
		Username:  username,
		Command:   command,
	})
	if len(bot.activityLog) > 100 {
		bot.activityLog = bot.activityLog[len(bot.activityLog)-100:]
	}
}

func (bot *Bot) GetActivityLog() []LogEntry {
	bot.mutex.RLock()
	defer bot.mutex.RUnlock()
	result := make([]LogEntry, len(bot.activityLog))
	copy(result, bot.activityLog)
	return result
}

func (bot *Bot) SetOnAuthFailed(fn func()) {
	bot.mutex.Lock()
	defer bot.mutex.Unlock()
	bot.onAuthFailed = fn
}

func (bot *Bot) UpdateConfig(cfg Config) {
	bot.mutex.Lock()
	defer bot.mutex.Unlock()
	if cfg.BotUsername != "" {
		bot.config.BotUsername = cfg.BotUsername
	}
	if cfg.Channel != "" {
		bot.config.Channel = cfg.Channel
	}
	if cfg.AccessToken != "" {
		bot.config.AccessToken = cfg.AccessToken
	}
}

func (bot *Bot) ReloadCommands() error {
	allCommands, err := database.GetAllCommands(bot.db)
	if err != nil {
		return err
	}

	commands := make(map[string]*database.Command)
	aliases := make(map[string]string)
	var keywords []*database.Command
	var keywordRegexps []*regexp.Regexp

	for i := range allCommands {
		command := &allCommands[i]
		switch command.Trigger {
		case "keyword":
			command.Name = strings.ToLower(command.Name)
			keywords = append(keywords, command)
			var pattern string
			if command.MatchStart {
				pattern = `(?i)^` + regexp.QuoteMeta(command.Name) + `\b`
			} else {
				pattern = `(?i)\b` + regexp.QuoteMeta(command.Name) + `\b`
			}
			keywordRegexps = append(keywordRegexps, regexp.MustCompile(pattern))
		default:
			commands[strings.ToLower(command.Name)] = command
			for _, alias := range command.Aliases {
				aliases[strings.ToLower(alias)] = strings.ToLower(command.Name)
			}
		}
	}

	bot.mutex.Lock()
	bot.commands = commands
	bot.aliases = aliases
	bot.keywords = keywords
	bot.keywordRegexps = keywordRegexps
	bot.mutex.Unlock()

	return nil
}

func (bot *Bot) Start(ctx context.Context) {
	if err := bot.ReloadCommands(); err != nil {
		slog.Error("failed to load commands", "error", err)
	}

	const (
		initialBackoff = 2 * time.Second
		maxBackoff     = 2 * time.Minute
	)
	backoff := initialBackoff

	for {
		bot.mutex.RLock()
		stopped := bot.stopped
		bot.mutex.RUnlock()

		if stopped {
			backoff = initialBackoff
			select {
			case <-ctx.Done():
				return
			case <-bot.startCh:
			}
			continue
		}

		bot.mutex.RLock()
		accessToken := bot.config.AccessToken
		channel := bot.config.Channel
		bot.mutex.RUnlock()

		if accessToken == "" || channel == "" {
			if accessToken == "" {
				slog.Warn("bot access token not set, visit the dashboard to authenticate")
			} else {
				slog.Warn("twitch channel not set")
			}
			backoff = initialBackoff
			select {
			case <-ctx.Done():
				return
			case <-bot.restartCh:
			case <-bot.startCh:
			}
			continue
		}

		wasConnected := bot.connect(ctx)

		select {
		case <-ctx.Done():
			return
		default:
		}

		// If the connection was healthy before dropping, reset backoff.
		if wasConnected {
			backoff = initialBackoff
		}

		// If a planned restart is already queued (token refresh, config change),
		// drain it and retry immediately without waiting.
		restarted := false
		select {
		case <-bot.restartCh:
			restarted = true
		default:
		}
		select {
		case <-bot.startCh:
			restarted = true
		default:
		}
		if restarted {
			backoff = initialBackoff
			continue
		}

		slog.Info("reconnecting after error", "delay", backoff)
		timer := time.NewTimer(backoff)
		backoff = min(backoff*2, maxBackoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-bot.restartCh:
			timer.Stop()
			backoff = initialBackoff
		case <-bot.startCh:
			timer.Stop()
			backoff = initialBackoff
		case <-timer.C:
		}
	}
}

func (bot *Bot) connect(ctx context.Context) (wasConnected bool) {
	bot.mutex.RLock()
	botUsername := bot.config.BotUsername
	accessToken := bot.config.AccessToken
	channel := bot.config.Channel
	bot.mutex.RUnlock()

	connCtx, connCancel := context.WithCancel(ctx)

	bot.mutex.Lock()
	bot.connCancel = connCancel
	bot.mutex.Unlock()

	defer func() {
		connCancel()
		bot.mutex.Lock()
		bot.connected = false
		bot.client = nil
		bot.connCancel = nil
		bot.mutex.Unlock()
	}()

	client := twitch.NewClient(botUsername, "oauth:"+accessToken)
	client.Join(channel)

	bot.mutex.Lock()
	bot.client = client
	bot.mutex.Unlock()

	client.OnConnect(func() {
		wasConnected = true
		bot.mutex.Lock()
		bot.connected = true
		bot.mutex.Unlock()
		slog.Info("connected to twitch", "channel", channel)
	})

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		bot.handleMessage(message)
	})

	go func() {
		<-connCtx.Done()
		client.Disconnect()
	}()

	if err := client.Connect(); err != nil {
		if errors.Is(err, twitch.ErrLoginAuthenticationFailed) {
			slog.Warn("twitch auth failed, attempting token refresh")
			bot.mutex.RLock()
			fn := bot.onAuthFailed
			bot.mutex.RUnlock()
			if fn != nil {
				fn()
			}
		} else {
			slog.Error("twitch connection error", "error", err)
		}
	}
	return
}

func (bot *Bot) Reconnect(accessToken string) {
	bot.mutex.Lock()
	bot.config.AccessToken = accessToken
	wasStopped := bot.stopped
	cancel := bot.connCancel
	bot.mutex.Unlock()

	if cancel != nil {
		cancel()
	}

	if !wasStopped {
		select {
		case bot.restartCh <- struct{}{}:
		default:
		}
	}
}

func (bot *Bot) handleMessage(message twitch.PrivateMessage) {
	bot.mutex.RLock()
	botUsername := bot.config.BotUsername
	bot.mutex.RUnlock()

	if strings.EqualFold(message.User.Name, botUsername) {
		return
	}

	text := strings.TrimSpace(message.Message)

	if strings.HasPrefix(text, "!") {
		bot.LogCommand(message.User.DisplayName, text)
		bot.handleCommand(message, text)
		return
	}

	bot.handleKeywordMatch(message, text)
}
