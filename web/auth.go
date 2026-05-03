package web

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mayor-wren/porygon/bot"
	porygonAuth "github.com/mayor-wren/porygon/auth"
)

func (server *Server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	accountType := r.PathValue("account")
	if accountType != "bot" && accountType != "streamer" {
		http.Error(w, "invalid account type", http.StatusBadRequest)
		return
	}

	if r.URL.Query().Get("same") == "true" {
		accountType = "both"
	}

	server.mu.RLock()
	clientID := server.config.TwitchClientID
	httpAddr := server.config.HTTPAddr
	server.mu.RUnlock()

	authorizeURL := porygonAuth.BuildAuthorizeURL(clientID, httpAddr, accountType)
	http.Redirect(w, r, authorizeURL, http.StatusFound)
}

func (server *Server) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || (state != "bot" && state != "streamer" && state != "both") {
		http.Error(w, "invalid callback", http.StatusBadRequest)
		return
	}

	server.mu.RLock()
	clientID := server.config.TwitchClientID
	clientSecret := server.config.TwitchClientSecret
	httpAddr := server.config.HTTPAddr
	server.mu.RUnlock()

	token, err := porygonAuth.ExchangeCode(clientID, clientSecret, httpAddr, code)
	if err != nil {
		slog.Error("token exchange failed", "error", err)
		http.Error(w, "token exchange failed", http.StatusInternalServerError)
		return
	}

	accounts := []string{state}
	if state == "both" {
		accounts = []string{"bot", "streamer"}
	}

	for _, account := range accounts {
		if err := server.saveAccountToken(account, token); err != nil {
			slog.Error("failed to save token", "account", account, "error", err)
			http.Error(w, "failed to save token", http.StatusInternalServerError)
			return
		}
	}

	slog.Info("oauth token saved", "account", state)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (server *Server) saveAccountToken(accountType string, token *porygonAuth.TokenResponse) error {
	if err := porygonAuth.SaveTokenToEnvFile(accountType, token); err != nil {
		return err
	}

	server.mu.Lock()
	switch accountType {
	case "bot":
		server.config.BotAccessToken = token.AccessToken
		server.config.BotRefreshToken = token.RefreshToken
	case "streamer":
		server.config.StreamerAccessToken = token.AccessToken
		server.config.StreamerRefreshToken = token.RefreshToken
	}
	server.mu.Unlock()

	if accountType == "bot" {
		server.bot.Reconnect(token.AccessToken)
	}

	return nil
}

func (server *Server) handleSaveSetup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}
	clientID := strings.TrimSpace(r.FormValue("client_id"))
	clientSecret := strings.TrimSpace(r.FormValue("client_secret"))
	botUsername := strings.TrimSpace(r.FormValue("bot_username"))
	streamerUsername := strings.TrimSpace(r.FormValue("streamer_username"))

	if clientID == "" || botUsername == "" || streamerUsername == "" {
		http.Error(w, "all fields are required", http.StatusBadRequest)
		return
	}

	if clientSecret == "" {
		server.mu.RLock()
		clientSecret = server.config.TwitchClientSecret
		server.mu.RUnlock()
	}
	if clientSecret == "" {
		http.Error(w, "client secret is required", http.StatusBadRequest)
		return
	}

	err := porygonAuth.UpdateEnvFile(".env", map[string]string{
		"TWITCH_CLIENT_ID":     clientID,
		"TWITCH_CLIENT_SECRET": clientSecret,
		"TWITCH_BOT_USERNAME":  botUsername,
		"TWITCH_CHANNEL":       streamerUsername,
	})
	if err != nil {
		slog.Error("failed to save config", "error", err)
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}

	server.mu.Lock()
	server.config.TwitchClientID = clientID
	server.config.TwitchClientSecret = clientSecret
	server.config.BotUsername = botUsername
	server.config.Channel = streamerUsername
	server.mu.Unlock()

	server.bot.UpdateConfig(bot.Config{
		BotUsername: botUsername,
		Channel:     streamerUsername,
	})

	slog.Info("config saved", "username", botUsername, "channel", streamerUsername)

	redirect := "/"
	if r.FormValue("_redirect") == "/settings" {
		redirect = "/settings"
	}
	http.Redirect(w, r, redirect, http.StatusFound)
}

func (server *Server) refreshBotToken() {
	server.mu.RLock()
	refreshToken := server.config.BotRefreshToken
	clientID := server.config.TwitchClientID
	clientSecret := server.config.TwitchClientSecret
	server.mu.RUnlock()

	if refreshToken == "" || clientID == "" || clientSecret == "" {
		return
	}

	token, err := porygonAuth.RefreshToken(clientID, clientSecret, refreshToken)
	if err != nil {
		slog.Error("failed to refresh bot token", "error", err)
		return
	}

	if err := server.saveAccountToken("bot", token); err != nil {
		slog.Error("failed to save refreshed bot token", "error", err)
		return
	}

	slog.Info("bot token refreshed")
	server.bot.Reconnect(token.AccessToken)
}

func (server *Server) StartTokenRefresh(ctx context.Context) {
	server.bot.SetOnAuthFailed(server.refreshBotToken)

	ticker := time.NewTicker(3 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			server.refreshBotToken()
		}
	}
}
