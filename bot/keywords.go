package bot

import (
	"time"

	"github.com/gempir/go-twitch-irc/v4"
)

func (bot *Bot) handleKeywordMatch(message twitch.PrivateMessage, text string) {
	bot.mutex.RLock()
	var matchedIdx int = -1
	for i, re := range bot.keywordRegexps {
		if !bot.keywords[i].Enabled {
			continue
		}
		if re.MatchString(text) {
			matchedIdx = i
			break
		}
	}
	bot.mutex.RUnlock()

	if matchedIdx == -1 {
		return
	}

	bot.mutex.RLock()
	keyword := bot.keywords[matchedIdx]
	bot.mutex.RUnlock()

	if keyword.Cooldown > 0 {
		bot.mutex.Lock()
		last, ok := bot.lastFired[keyword.Name]
		if ok && time.Since(last) < time.Duration(keyword.Cooldown)*time.Second {
			bot.mutex.Unlock()
			return
		}
		bot.lastFired[keyword.Name] = time.Now()
		bot.mutex.Unlock()
	}

	bot.client.Say(message.Channel, keyword.Response)
	bot.LogCommand(message.User.DisplayName, keyword.Name)
}
