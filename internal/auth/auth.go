// Package auth provides Google OAuth2 authentication for mailbeads.
//
// It reads the same credentials.json and token.json files used by the
// Python google-auth library, so existing tokens work without re-authentication.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gmail "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Default scopes matching the Python scripts.
var DefaultScopes = []string{
	"https://www.googleapis.com/auth/gmail.readonly",
	"https://www.googleapis.com/auth/gmail.compose",
	"https://www.googleapis.com/auth/gmail.modify",
}

// pythonToken represents the token.json format written by Python's google-auth library.
type pythonToken struct {
	Token        string   `json:"token"`
	RefreshToken string   `json:"refresh_token"`
	TokenURI     string   `json:"token_uri"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	Scopes       []string `json:"scopes"`
	Expiry       string   `json:"expiry"`
}

// LoadGmailService returns an authenticated Gmail API service for the given account.
// credentialsPath should point to the credentials.json file (e.g., "account@example.com/credentials.json").
func LoadGmailService(ctx context.Context, credentialsPath string) (*gmail.Service, error) {
	client, err := getClient(ctx, credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("get oauth client: %w", err)
	}
	return gmail.NewService(ctx, option.WithHTTPClient(client))
}

// getClient returns an authenticated HTTP client by loading the OAuth config
// from credentials.json and the token from token.json.
func getClient(ctx context.Context, credentialsPath string) (*http.Client, error) {
	config, err := loadOAuthConfig(credentialsPath)
	if err != nil {
		return nil, err
	}

	tokenPath := filepath.Join(filepath.Dir(credentialsPath), "token.json")
	token, err := loadPythonToken(tokenPath, config)
	if err != nil {
		return nil, fmt.Errorf("load token from %s: %w", tokenPath, err)
	}

	// Use a token source that auto-refreshes and save the refreshed token.
	ts := config.TokenSource(ctx, token)
	newToken, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("refresh token: %w", err)
	}

	// If token was refreshed, save it back in Python format.
	if newToken.AccessToken != token.AccessToken {
		if saveErr := savePythonToken(tokenPath, newToken, config); saveErr != nil {
			// Non-fatal: log but don't fail.
			fmt.Fprintf(os.Stderr, "warning: could not save refreshed token: %v\n", saveErr)
		}
	}

	return oauth2.NewClient(ctx, ts), nil
}

// loadOAuthConfig reads credentials.json and returns an OAuth2 config.
func loadOAuthConfig(credentialsPath string) (*oauth2.Config, error) {
	data, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("read credentials from %s: %w", credentialsPath, err)
	}

	config, err := google.ConfigFromJSON(data, DefaultScopes...)
	if err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}

	return config, nil
}

// loadPythonToken reads a token.json file in Python google-auth format
// and converts it to a Go oauth2.Token.
func loadPythonToken(tokenPath string, config *oauth2.Config) (*oauth2.Token, error) {
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("read token: %w", err)
	}

	var pt pythonToken
	if err := json.Unmarshal(data, &pt); err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	// Parse expiry time. Python writes ISO 8601 with microseconds.
	var expiry time.Time
	if pt.Expiry != "" {
		for _, layout := range []string{
			"2006-01-02T15:04:05.999999Z",
			"2006-01-02T15:04:05Z",
			time.RFC3339,
			time.RFC3339Nano,
		} {
			if t, err := time.Parse(layout, pt.Expiry); err == nil {
				expiry = t
				break
			}
		}
	}

	return &oauth2.Token{
		AccessToken:  pt.Token,
		RefreshToken: pt.RefreshToken,
		TokenType:    "Bearer",
		Expiry:       expiry,
	}, nil
}

// savePythonToken writes a token back in the Python google-auth format
// so the Python scripts can still use it.
func savePythonToken(tokenPath string, token *oauth2.Token, config *oauth2.Config) error {
	pt := pythonToken{
		Token:        token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenURI:     config.Endpoint.TokenURL,
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Scopes:       DefaultScopes,
		Expiry:       token.Expiry.UTC().Format("2006-01-02T15:04:05.999999Z"),
	}

	data, err := json.MarshalIndent(pt, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(tokenPath, data, 0o600); err != nil {
		return err
	}
	return nil
}
