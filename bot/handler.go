package bot

import (
	"strings"
	"time"

	"github.com/gempir/go-twitch-irc/v4"
	"github.com/mayor-wren/porygon/database"
)

func (bot *Bot) handleCommand(message twitch.PrivateMessage, text string) {
	withoutPrefix := text[1:]
	parts := strings.SplitN(withoutPrefix, " ", 2)
	commandName := strings.ToLower(parts[0])
	arguments := ""
	if len(parts) > 1 {
		arguments = parts[1]
	}

	switch commandName {
	case "quote":
		bot.handleQuote(message, arguments)
		return
	case "addquote":
		bot.handleAddQuote(message, arguments)
		return
	case "delquote":
		bot.handleDelQuote(message, arguments)
		return
	}

	bot.mutex.RLock()
	command := bot.lookupCommand(commandName)
	bot.mutex.RUnlock()

	if command == nil || !command.Enabled {
		return
	}

	if command.Cooldown > 0 {
		bot.mutex.Lock()
		last, ok := bot.lastFired[commandName]
		if ok && time.Since(last) < time.Duration(command.Cooldown)*time.Second {
			bot.mutex.Unlock()
			return
		}
		bot.lastFired[commandName] = time.Now()
		bot.mutex.Unlock()
	}

	// bot.client is safe without the lock: it is only set to nil after
	// client.Connect() returns, at which point no more callbacks can fire.
	bot.client.Say(message.Channel, command.Response)
}

func (bot *Bot) lookupCommand(name string) *database.Command {
	if command, exists := bot.commands[name]; exists {
		return command
	}
	if primaryName, exists := bot.aliases[name]; exists {
		if command, exists := bot.commands[primaryName]; exists {
			return command
		}
	}
	return nil
}
