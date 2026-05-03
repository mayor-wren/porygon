package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildAuthorizeURL(t *testing.T) {
	url := BuildAuthorizeURL("test-client-id", ":8080", "bot")

	if !strings.Contains(url, "client_id=test-client-id") {
		t.Error("expected URL to contain client_id")
	}
	if !strings.Contains(url, "state=bot") {
		t.Error("expected URL to contain state=bot")
	}
	if !strings.Contains(url, "chat%3Aread") {
		t.Error("expected URL to contain chat:read scope")
	}
}

func TestBuildAuthorizeURLStreamer(t *testing.T) {
	url := BuildAuthorizeURL("test-client-id", ":8080", "streamer")

	if !strings.Contains(url, "state=streamer") {
		t.Error("expected URL to contain state=streamer")
	}
	if !strings.Contains(url, "channel%3Aread%3Asubscriptions") {
		t.Error("expected URL to contain channel:read:subscriptions scope for streamer")
	}
}

func TestUpdateEnvFileCreatesNewFile(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	err := UpdateEnvFile(envPath, map[string]string{
		"FOO": "bar",
		"BAZ": "qux",
	})
	if err != nil {
		t.Fatalf("failed to update env file: %v", err)
	}

	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("failed to read env file: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "FOO=bar") {
		t.Error("expected FOO=bar in env file")
	}
	if !strings.Contains(text, "BAZ=qux") {
		t.Error("expected BAZ=qux in env file")
	}
}

func TestUpdateEnvFileUpdatesExistingKeys(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	os.WriteFile(envPath, []byte("FOO=old\nBAR=keep\n"), 0600)

	err := UpdateEnvFile(envPath, map[string]string{
		"FOO": "new",
	})
	if err != nil {
		t.Fatalf("failed to update env file: %v", err)
	}

	content, _ := os.ReadFile(envPath)
	text := string(content)

	if !strings.Contains(text, "FOO=new") {
		t.Error("expected FOO=new")
	}
	if !strings.Contains(text, "BAR=keep") {
		t.Error("expected BAR=keep to be preserved")
	}
	if strings.Contains(text, "FOO=old") {
		t.Error("old value should have been replaced")
	}
}

func TestUpdateEnvFilePreservesComments(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	os.WriteFile(envPath, []byte("# This is a comment\nFOO=old\n"), 0600)

	UpdateEnvFile(envPath, map[string]string{"FOO": "new"})

	content, _ := os.ReadFile(envPath)
	text := string(content)

	if !strings.Contains(text, "# This is a comment") {
		t.Error("comment should be preserved")
	}
}

func TestSaveTokenToEnvFileBot(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	token := &TokenResponse{
		AccessToken:  "access123",
		RefreshToken: "refresh456",
	}

	err := SaveTokenToEnvFile("bot", token)
	if err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	content, _ := os.ReadFile(".env")
	text := string(content)

	if !strings.Contains(text, "TWITCH_BOT_ACCESS_TOKEN=access123") {
		t.Error("expected bot access token")
	}
	if !strings.Contains(text, "TWITCH_BOT_REFRESH_TOKEN=refresh456") {
		t.Error("expected bot refresh token")
	}
}

func TestSaveTokenToEnvFileInvalidType(t *testing.T) {
	err := SaveTokenToEnvFile("invalid", &TokenResponse{})
	if err == nil {
		t.Error("expected error for invalid account type")
	}
}
