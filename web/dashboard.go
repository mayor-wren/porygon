package web

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (server *Server) isSetupComplete() bool {
	server.mu.RLock()
	defer server.mu.RUnlock()
	return server.config.TwitchClientID != "" &&
		server.config.TwitchClientSecret != "" &&
		server.config.BotAccessToken != "" &&
		server.config.BotUsername != "" &&
		server.config.Channel != ""
}

func (server *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	server.mu.RLock()
	hasSetupInfo := server.config.TwitchClientID != "" &&
		server.config.TwitchClientSecret != "" &&
		server.config.BotUsername != "" &&
		server.config.Channel != ""
	data := map[string]any{
		"HasSetupInfo":     hasSetupInfo,
		"HasBotToken":      server.config.BotAccessToken != "",
		"HasStreamerToken": server.config.StreamerAccessToken != "",
		"BotUsername":      server.config.BotUsername,
		"Channel":          server.config.Channel,
		"ClientID":         server.config.TwitchClientID,
		"CallbackURL":      fmt.Sprintf("http://localhost%s/auth/callback", server.config.HTTPAddr),
	}
	server.mu.RUnlock()

	data["BotConnected"] = server.bot.IsConnected()
	data["BotStopped"] = server.bot.IsStopped()

	server.renderTemplate(w, "dashboard.html", data)
}

func (server *Server) handleSettingsPage(w http.ResponseWriter, r *http.Request) {
	if !server.isSetupComplete() {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	server.mu.RLock()
	data := map[string]any{
		"Channel":          server.config.Channel,
		"BotUsername":      server.config.BotUsername,
		"ClientID":         server.config.TwitchClientID,
		"HasBotToken":      server.config.BotAccessToken != "",
		"HasStreamerToken": server.config.StreamerAccessToken != "",
		"CallbackURL":      fmt.Sprintf("http://localhost%s/auth/callback", server.config.HTTPAddr),
	}
	server.mu.RUnlock()

	server.renderTemplate(w, "settings.html", data)
}

func (server *Server) handleBotStart(w http.ResponseWriter, r *http.Request) {
	server.bot.StartBot()
	http.Redirect(w, r, "/", http.StatusFound)
}

func (server *Server) handleBotStop(w http.ResponseWriter, r *http.Request) {
	server.bot.StopBot()
	http.Redirect(w, r, "/", http.StatusFound)
}

func (server *Server) handleBotStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Connected bool `json:"connected"`
		Stopped   bool `json:"stopped"`
	}{
		Connected: server.bot.IsConnected(),
		Stopped:   server.bot.IsStopped(),
	})
}

func (server *Server) handleBotLog(w http.ResponseWriter, r *http.Request) {
	entries := server.bot.GetActivityLog()

	type logEntry struct {
		Time    string `json:"time"`
		User    string `json:"user"`
		Command string `json:"command"`
	}

	result := make([]logEntry, len(entries))
	for i, e := range entries {
		result[len(entries)-1-i] = logEntry{
			Time:    e.Timestamp.Format("15:04:05"),
			User:    e.Username,
			Command: e.Command,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
