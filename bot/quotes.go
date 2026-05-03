package bot

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/gempir/go-twitch-irc/v4"
	"github.com/mayor-wren/porygon/database"
)

func (bot *Bot) handleQuote(message twitch.PrivateMessage, arguments string) {
	arguments = strings.TrimSpace(arguments)

	if arguments == "" {
		quote, err := database.GetRandomQuote(bot.db)
		if err == sql.ErrNoRows {
			bot.client.Say(message.Channel, "No quotes yet!")
			return
		}
		if err != nil {
			slog.Error("failed to get random quote", "error", err)
			return
		}
		bot.client.Say(message.Channel, fmt.Sprintf("Quote #%d: %s", quote.ID, quote.Text))
		return
	}

	quoteID, err := strconv.Atoi(arguments)
	if err != nil {
		bot.client.Say(message.Channel, "Usage: !quote or !quote <number>")
		return
	}

	quote, err := database.GetQuote(bot.db, quoteID)
	if err == sql.ErrNoRows {
		bot.client.Say(message.Channel, fmt.Sprintf("Quote #%d not found.", quoteID))
		return
	}
	if err != nil {
		slog.Error("failed to get quote", "error", err)
		return
	}
	bot.client.Say(message.Channel, fmt.Sprintf("Quote #%d: %s", quote.ID, quote.Text))
}

func (bot *Bot) handleAddQuote(message twitch.PrivateMessage, arguments string) {
	_, isModerator := message.User.Badges["moderator"]
	_, isBroadcaster := message.User.Badges["broadcaster"]
	if !isModerator && !isBroadcaster {
		bot.client.Say(message.Channel, "Only moderators can add quotes.")
		return
	}

	quoteText := strings.TrimSpace(arguments)
	if quoteText == "" {
		bot.client.Say(message.Channel, "Usage: !addquote <text>")
		return
	}

	quoteID, err := database.CreateQuote(bot.db, quoteText, message.User.Name)
	if err != nil {
		slog.Error("failed to add quote", "error", err)
		return
	}
	bot.client.Say(message.Channel, fmt.Sprintf("Quote #%d added!", quoteID))
}
