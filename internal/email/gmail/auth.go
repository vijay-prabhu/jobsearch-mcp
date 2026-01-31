package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

// GmailScopes defines the OAuth scopes required
var GmailScopes = []string{
	gmail.GmailReadonlyScope,
}

// loadCredentials loads OAuth config from credentials file
func loadCredentials(credPath string) (*oauth2.Config, error) {
	data, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w\n\nTo set up Gmail API:\n1. Go to https://console.cloud.google.com/\n2. Create a project and enable Gmail API\n3. Create OAuth 2.0 credentials (Desktop app)\n4. Download and save to: %s", err, credPath)
	}

	config, err := google.ConfigFromJSON(data, GmailScopes...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return config, nil
}

// loadToken loads a saved OAuth token
func loadToken(tokenPath string) (*oauth2.Token, error) {
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, err
	}

	token := &oauth2.Token{}
	if err := json.Unmarshal(data, token); err != nil {
		return nil, err
	}

	return token, nil
}

// saveToken saves an OAuth token to file
func saveToken(tokenPath string, token *oauth2.Token) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(tokenPath, data, 0600)
}

// getTokenFromWeb performs the OAuth flow via browser
func getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	// Use a random state for security
	state := fmt.Sprintf("%d", time.Now().UnixNano())

	// Create a channel to receive the auth code
	codeChan := make(chan string)
	errChan := make(chan error)

	// Start a local server to receive the callback
	server := &http.Server{Addr: "localhost:8080"}

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errChan <- fmt.Errorf("invalid state parameter")
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no code in callback")
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><body><h1>Authentication successful!</h1><p>You can close this window.</p></body></html>`)
		codeChan <- code
	})

	// Start server in background
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Configure for localhost callback
	config.RedirectURL = "http://localhost:8080/callback"

	// Generate auth URL
	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	fmt.Println("Opening browser for Google authentication...")
	fmt.Println("If browser doesn't open, visit this URL:")
	fmt.Println(authURL)
	fmt.Println()

	// Try to open browser
	openBrowser(authURL)

	// Wait for code or error
	var code string
	select {
	case code = <-codeChan:
	case err := <-errChan:
		_ = server.Shutdown(ctx)
		return nil, err
	case <-ctx.Done():
		_ = server.Shutdown(ctx)
		return nil, ctx.Err()
	case <-time.After(5 * time.Minute):
		_ = server.Shutdown(ctx)
		return nil, fmt.Errorf("authentication timeout")
	}

	// Shutdown server
	_ = server.Shutdown(ctx)

	// Exchange code for token
	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	return token, nil
}

// openBrowser opens the URL in the default browser
func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}

	_ = cmd.Start()
}

// getClient returns an authenticated HTTP client
func getClient(ctx context.Context, config *oauth2.Config, tokenPath string) (*http.Client, error) {
	token, err := loadToken(tokenPath)
	if err != nil {
		// Need to authenticate
		token, err = getTokenFromWeb(ctx, config)
		if err != nil {
			return nil, err
		}

		if err := saveToken(tokenPath, token); err != nil {
			return nil, fmt.Errorf("failed to save token: %w", err)
		}

		fmt.Println("Authentication successful!")
	}

	// Token source will auto-refresh expired tokens
	tokenSource := config.TokenSource(ctx, token)

	// Save refreshed token if it changed
	newToken, err := tokenSource.Token()
	if err == nil && newToken.AccessToken != token.AccessToken {
		_ = saveToken(tokenPath, newToken)
	}

	return oauth2.NewClient(ctx, tokenSource), nil
}
