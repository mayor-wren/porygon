package web

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mayor-wren/porygon/database"
)

func (server *Server) handleQuotesList(w http.ResponseWriter, r *http.Request) {
	quotes, err := database.GetAllQuotes(server.db)
	if err != nil {
		slog.Error("failed to list quotes", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	server.renderTemplate(w, "quotes.html", map[string]any{
		"Quotes": quotes,
	})
}

func (server *Server) handleQuoteNew(w http.ResponseWriter, r *http.Request) {
	server.renderTemplate(w, "quote_form.html", map[string]any{
		"IsNew": true,
	})
}

func (server *Server) handleQuoteCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}
	text := strings.TrimSpace(r.FormValue("text"))
	addedBy := strings.TrimSpace(r.FormValue("added_by"))

	if text == "" {
		http.Error(w, "text is required", http.StatusBadRequest)
		return
	}
	if addedBy == "" {
		addedBy = "dashboard"
	}

	if _, err := database.CreateQuote(server.db, text, addedBy); err != nil {
		slog.Error("failed to create quote", "error", err)
		http.Error(w, "failed to create quote", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/quotes", http.StatusFound)
}

func (server *Server) handleQuoteEdit(w http.ResponseWriter, r *http.Request) {
	quoteID, err := pathID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	quote, err := database.GetQuote(server.db, quoteID)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "quote not found", http.StatusNotFound)
		return
	}
	if err != nil {
		slog.Error("failed to get quote", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	server.renderTemplate(w, "quote_form.html", map[string]any{
		"IsNew": false,
		"Quote": quote,
	})
}

func (server *Server) handleQuoteUpdate(w http.ResponseWriter, r *http.Request) {
	quoteID, err := pathID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}
	text := strings.TrimSpace(r.FormValue("text"))

	if text == "" {
		http.Error(w, "text is required", http.StatusBadRequest)
		return
	}

	if err := database.UpdateQuote(server.db, quoteID, text); err != nil {
		slog.Error("failed to update quote", "error", err)
		http.Error(w, "failed to update quote", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/quotes", http.StatusFound)
}

func (server *Server) handleQuoteDelete(w http.ResponseWriter, r *http.Request) {
	quoteID, err := pathID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := database.DeleteQuote(server.db, quoteID); err != nil {
		slog.Error("failed to delete quote", "error", err)
		http.Error(w, "failed to delete quote", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/quotes", http.StatusFound)
}
