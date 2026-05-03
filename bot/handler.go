package bot

import (
	"strings"

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
	}

	bot.mutex.RLock()
	command := bot.lookupCommand(commandName)
	bot.mutex.RUnlock()

	// bot.client is safe to use without the lock here: it is only set to nil
	// after client.Connect() returns, at which point no more callbacks can fire.
	if command != nil && command.Enabled {
		bot.client.Say(message.Channel, command.Response)
	}
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
