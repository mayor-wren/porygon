package bot

import (
	"strings"

	"github.com/gempir/go-twitch-irc/v4"
	"github.com/mayor-wren/porygon/database"
)

func (bot *Bot) handleKeywordMatch(message twitch.PrivateMessage, text string) {
	lowercaseText := strings.ToLower(text)

	bot.mutex.RLock()
	var matched *database.Command
	for _, keyword := range bot.keywords {
		if !keyword.Enabled {
			continue
		}
		if strings.Contains(lowercaseText, keyword.Name) {
			matched = keyword
			break
		}
	}
	bot.mutex.RUnlock()

	if matched != nil {
		bot.client.Say(message.Channel, matched.Response)
		bot.LogCommand(message.User.DisplayName, matched.Name)
	}
}
