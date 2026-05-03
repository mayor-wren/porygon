package main

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/mayor-wren/porygon/bot"
	"github.com/mayor-wren/porygon/database"
	"github.com/mayor-wren/porygon/web"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

func main() {
	godotenv.Load()

	httpAddr := envOrDefault("HTTP_ADDR", ":7310")
	databasePath := envOrDefault("DATABASE_PATH", "./porygon.db")

	db, err := database.Open(databasePath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	chatBot := bot.New(&bot.Config{
		BotUsername:  os.Getenv("TWITCH_BOT_USERNAME"),
		Channel:     os.Getenv("TWITCH_CHANNEL"),
		AccessToken: os.Getenv("TWITCH_BOT_ACCESS_TOKEN"),
	}, db)

	templates, _ := fs.Sub(templateFS, "templates")
	static, _ := fs.Sub(staticFS, "static")

	webServer := web.NewServer(&web.Config{
		TwitchClientID:       os.Getenv("TWITCH_CLIENT_ID"),
		TwitchClientSecret:   os.Getenv("TWITCH_CLIENT_SECRET"),
		HTTPAddr:             httpAddr,
		BotUsername:           os.Getenv("TWITCH_BOT_USERNAME"),
		Channel:              os.Getenv("TWITCH_CHANNEL"),
		BotAccessToken:       os.Getenv("TWITCH_BOT_ACCESS_TOKEN"),
		BotRefreshToken:      os.Getenv("TWITCH_BOT_REFRESH_TOKEN"),
		StreamerAccessToken:  os.Getenv("TWITCH_STREAMER_ACCESS_TOKEN"),
		StreamerRefreshToken: os.Getenv("TWITCH_STREAMER_REFRESH_TOKEN"),
	}, db, chatBot, templates, static)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go chatBot.Start(ctx)
	go webServer.StartTokenRefresh(ctx)
	go func() {
		slog.Info("dashboard available", "addr", "http://localhost"+httpAddr)
		if err := webServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("web server error", "error", err)
			cancel()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	webServer.Shutdown(context.Background())
}

func envOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
