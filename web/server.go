package web

import (
	"database/sql"
	"html/template"
	"io/fs"
	"net/http"
	"strconv"
	"sync"

	"github.com/mayor-wren/porygon/bot"
)

type Config struct {
	TwitchClientID       string
	TwitchClientSecret   string
	HTTPAddr             string
	BotUsername          string
	Channel              string
	BotAccessToken       string
	BotRefreshToken      string
	StreamerAccessToken  string
	StreamerRefreshToken string
}

type Server struct {
	config    *Config
	db        *sql.DB
	bot       *bot.Bot
	mu        sync.RWMutex
	templates map[string]*template.Template
	*http.Server
}

func NewServer(config *Config, db *sql.DB, chatBot *bot.Bot, templateFS, staticFS fs.FS) *Server {
	server := &Server{
		config: config,
		db:     db,
		bot:    chatBot,
	}

	server.loadTemplates(templateFS)

	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFS)))

	mux.HandleFunc("GET /{$}", server.handleDashboard)

	mux.HandleFunc("GET /auth/login/{account}", server.handleAuthLogin)
	mux.HandleFunc("GET /auth/callback", server.handleAuthCallback)
	mux.HandleFunc("POST /setup", server.handleSaveSetup)

	mux.HandleFunc("POST /bot/start", server.handleBotStart)
	mux.HandleFunc("POST /bot/stop", server.handleBotStop)
	mux.HandleFunc("GET /bot/status", server.handleBotStatus)
	mux.HandleFunc("GET /bot/log", server.handleBotLog)

	mux.HandleFunc("GET /settings", server.handleSettingsPage)

	mux.HandleFunc("GET /commands", server.handleCommandsList)
	mux.HandleFunc("GET /commands/new", server.handleCommandNew)
	mux.HandleFunc("POST /commands", server.handleCommandCreate)
	mux.HandleFunc("GET /commands/{id}/edit", server.handleCommandEdit)
	mux.HandleFunc("POST /commands/{id}", server.handleCommandUpdate)
	mux.HandleFunc("POST /commands/{id}/delete", server.handleCommandDelete)

	mux.HandleFunc("GET /quotes", server.handleQuotesList)
	mux.HandleFunc("GET /quotes/new", server.handleQuoteNew)
	mux.HandleFunc("POST /quotes", server.handleQuoteCreate)
	mux.HandleFunc("GET /quotes/{id}/edit", server.handleQuoteEdit)
	mux.HandleFunc("POST /quotes/{id}", server.handleQuoteUpdate)
	mux.HandleFunc("POST /quotes/{id}/delete", server.handleQuoteDelete)

	server.Server = &http.Server{
		Addr:    config.HTTPAddr,
		Handler: mux,
	}

	return server
}

func (server *Server) loadTemplates(templateFS fs.FS) {
	pageFiles := []string{
		"dashboard.html",
		"commands.html",
		"command_form.html",
		"quotes.html",
		"quote_form.html",
		"settings.html",
	}

	server.templates = make(map[string]*template.Template)
	for _, page := range pageFiles {
		server.templates[page] = template.Must(template.ParseFS(templateFS, "layout.html", page))
	}
}

func pathID(r *http.Request) (int, error) {
	return strconv.Atoi(r.PathValue("id"))
}

func (server *Server) renderTemplate(w http.ResponseWriter, name string, data map[string]any) {
	tmpl, ok := server.templates[name]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}

	// Wrap data so we inject SetupComplete without mutating the caller's map
	wrapped := make(map[string]any, len(data)+1)
	for k, v := range data {
		wrapped[k] = v
	}
	if _, exists := wrapped["SetupComplete"]; !exists {
		wrapped["SetupComplete"] = server.isSetupComplete()
	}

	if err := tmpl.ExecuteTemplate(w, "layout", wrapped); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}
