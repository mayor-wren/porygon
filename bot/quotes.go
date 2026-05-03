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

func formatQuote(quote *database.Quote) string {
	return fmt.Sprintf(`%d. "%s" - %s`, quote.ID, quote.Text, quote.CreatedAt.Format("January 2, 2006"))
}

func (bot *Bot) handleQuote(message twitch.PrivateMessage, arguments string) {
	arguments = strings.TrimSpace(arguments)

	if arguments == "" {
		quote, err := database.GetRandomQuote(bot.db)
		if err == sql.ErrNoRows {
			bot.client.Say(message.Channel, "There are currently no saved quotes!")
			return
		}
		if err != nil {
			slog.Error("failed to get random quote", "error", err)
			return
		}
		bot.client.Say(message.Channel, formatQuote(quote))
		return
	}

	quoteID, err := strconv.Atoi(arguments)
	if err == nil {
		quote, err := database.GetQuote(bot.db, quoteID)
		if err == sql.ErrNoRows {
			return
		}
		if err != nil {
			slog.Error("failed to get quote", "error", err)
			return
		}
		bot.client.Say(message.Channel, formatQuote(quote))
		return
	}

	// Text search — fall back to random if no match, same as old bot behavior
	quote, err := database.SearchQuote(bot.db, arguments)
	if err == sql.ErrNoRows {
		quote, err = database.GetRandomQuote(bot.db)
		if err == sql.ErrNoRows {
			bot.client.Say(message.Channel, "There are currently no saved quotes!")
			return
		}
	}
	if err != nil {
		slog.Error("failed to search quotes", "error", err)
		return
	}
	bot.client.Say(message.Channel, formatQuote(quote))
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

	exists, err := database.QuoteExists(bot.db, quoteText)
	if err != nil {
		slog.Error("failed to check for duplicate quote", "error", err)
		return
	}
	if exists {
		bot.client.Say(message.Channel, fmt.Sprintf("@%s That quotation is already saved.", message.User.DisplayName))
		return
	}

	quoteID, err := database.CreateQuote(bot.db, quoteText, message.User.Name)
	if err != nil {
		slog.Error("failed to add quote", "error", err)
		return
	}
	bot.client.Say(message.Channel, fmt.Sprintf("Added quote %d", quoteID))
}

func (bot *Bot) handleDelQuote(message twitch.PrivateMessage, arguments string) {
	_, isModerator := message.User.Badges["moderator"]
	_, isBroadcaster := message.User.Badges["broadcaster"]
	if !isModerator && !isBroadcaster {
		bot.client.Say(message.Channel, "Only moderators can delete quotes.")
		return
	}

	quoteID, err := strconv.Atoi(strings.TrimSpace(arguments))
	if err != nil || quoteID < 1 {
		bot.client.Say(message.Channel, "Usage: !delquote <number>")
		return
	}

	if _, err := database.GetQuote(bot.db, quoteID); err == sql.ErrNoRows {
		bot.client.Say(message.Channel, fmt.Sprintf("Quote #%d not found.", quoteID))
		return
	}

	if err := database.DeleteQuote(bot.db, quoteID); err != nil {
		slog.Error("failed to delete quote", "error", err)
		return
	}
	bot.client.Say(message.Channel, fmt.Sprintf("Deleted quote %d.", quoteID))
}
