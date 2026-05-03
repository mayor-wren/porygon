package auth

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	TwitchAuthorizeURL = "https://id.twitch.tv/oauth2/authorize"
	TwitchTokenURL     = "https://id.twitch.tv/oauth2/token"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func BuildAuthorizeURL(clientID, httpAddr, accountType string) string {
	scopes := "chat:read chat:edit"
	if accountType == "streamer" || accountType == "both" {
		scopes = "chat:read chat:edit channel:read:subscriptions"
	}

	params := url.Values{
		"response_type": {"code"},
		"client_id":     {clientID},
		"redirect_uri":  {fmt.Sprintf("http://localhost%s/auth/callback", httpAddr)},
		"scope":         {scopes},
		"state":         {accountType},
		"force_verify":  {"true"},
	}

	return TwitchAuthorizeURL + "?" + params.Encode()
}

func ExchangeCode(clientID, clientSecret, httpAddr, code string) (*TokenResponse, error) {
	response, err := http.PostForm(TwitchTokenURL, url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {fmt.Sprintf("http://localhost%s/auth/callback", httpAddr)},
	})
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d", response.StatusCode)
	}

	var token TokenResponse
	if err := json.NewDecoder(response.Body).Decode(&token); err != nil {
		return nil, err
	}
	return &token, nil
}

func RefreshToken(clientID, clientSecret, refreshToken string) (*TokenResponse, error) {
	response, err := http.PostForm(TwitchTokenURL, url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	})
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed with status %d", response.StatusCode)
	}

	var token TokenResponse
	if err := json.NewDecoder(response.Body).Decode(&token); err != nil {
		return nil, err
	}
	return &token, nil
}

func SaveTokenToEnvFile(accountType string, token *TokenResponse) error {
	var accessKey, refreshKey string
	switch accountType {
	case "bot":
		accessKey = "TWITCH_BOT_ACCESS_TOKEN"
		refreshKey = "TWITCH_BOT_REFRESH_TOKEN"
	case "streamer":
		accessKey = "TWITCH_STREAMER_ACCESS_TOKEN"
		refreshKey = "TWITCH_STREAMER_REFRESH_TOKEN"
	default:
		return fmt.Errorf("unknown account type: %s", accountType)
	}

	return UpdateEnvFile(".env", map[string]string{
		accessKey:  token.AccessToken,
		refreshKey: token.RefreshToken,
	})
}

func UpdateEnvFile(path string, updates map[string]string) error {
	lines, err := readLines(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	remaining := make(map[string]string)
	for key, value := range updates {
		remaining[key] = value
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		if value, ok := remaining[key]; ok {
			lines[i] = key + "=" + value
			delete(remaining, key)
		}
	}

	for key, value := range remaining {
		lines = append(lines, key+"="+value)
	}

	return writeLines(path, lines)
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func writeLines(path string, lines []string) error {
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0600)
}
