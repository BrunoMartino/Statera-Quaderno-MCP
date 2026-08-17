package config

import (
	"encoding/base64"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"

	"woocommerce-store-mcp/internal/guard"
)

func testFactory(env map[string]string) *EnvConfigFactory {
	return NewEnvConfigFactory(func(k string) string { return env[k] }).withReadFile(func(string) ([]byte, error) {
		return nil, os.ErrNotExist
	})
}

func TestGetConfigMissingCredentialsReturnsAuthMissing(t *testing.T) {
	// Arrange
	f := testFactory(map[string]string{"WP_BASE_URL": "https://loja.example.com"})
	// Act
	cfg, err := f.GetConfig()
	// Assert
	if cfg != nil {
		t.Fatalf("expected nil config")
	}
	if !guard.IsCode(err, guard.AuthMissing) {
		t.Fatalf("got %v, want AUTH_MISSING", err)
	}
	if err != nil && strings.Contains(err.Error(), "xxxx") {
		t.Fatalf("error leaked a secret: %v", err)
	}
}

func TestGetConfigApplicationPasswordSetsBasicWithoutConsumerQuery(t *testing.T) {
	// Arrange
	f := testFactory(map[string]string{
		"WP_BASE_URL":        "https://loja.example.com/",
		"WP_APP_USER":        "mcp-content",
		"WP_APP_PASSWORD":    "test-app-password-123",
		"WC_CONSUMER_KEY":    "ck_should_not_win",
		"WC_CONSUMER_SECRET": "cs_should_not_win",
	})
	// Act
	cfg, err := f.GetConfig()
	// Assert
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if cfg.BaseURL != "https://loja.example.com" {
		t.Fatalf("BaseURL=%q", cfg.BaseURL)
	}
	req, _ := http.NewRequest(http.MethodGet, cfg.BaseURL+"/wp-json/wp/v2/posts", nil)
	cfg.Auth.Apply(req)
	if req.URL.Query().Get("consumer_key") != "" || req.URL.RawQuery != "" {
		t.Fatalf("consumer credentials in query: %s", req.URL.String())
	}
	user, pass, ok := req.BasicAuth()
	if !ok || user != "mcp-content" || pass != "test-app-password-123" {
		t.Fatalf("basic auth user=%q pass=%q ok=%v", user, pass, ok)
	}
	wantPrefix := "Basic " + base64.StdEncoding.EncodeToString([]byte("mcp-content:test-app-password-123"))
	if req.Header.Get("Authorization") != wantPrefix {
		t.Fatalf("Authorization mismatch")
	}
}

func TestGetConfigWooCommerceConsumerPairUsesBasic(t *testing.T) {
	// Arrange
	f := testFactory(map[string]string{
		"WP_BASE_URL":        "https://loja.example.com",
		"WC_CONSUMER_KEY":    "ck_test_key_123",
		"WC_CONSUMER_SECRET": "cs_test_secret_123",
	})
	// Act
	cfg, err := f.GetConfig()
	// Assert
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	req, _ := http.NewRequest(http.MethodGet, cfg.BaseURL+"/wp-json/wc/v3/products", nil)
	cfg.Auth.Apply(req)
	if strings.Contains(req.URL.String(), "consumer_key") || strings.Contains(req.URL.String(), "consumer_secret") {
		t.Fatalf("consumer in URL: %s", req.URL.String())
	}
	user, pass, ok := req.BasicAuth()
	if !ok || user != "ck_test_key_123" || pass != "cs_test_secret_123" {
		t.Fatalf("basic auth user=%q pass=%q ok=%v", user, pass, ok)
	}
}

func TestGetConfigHTTPBaseURLIsAuthMissing(t *testing.T) {
	// Arrange
	f := testFactory(map[string]string{
		"WP_BASE_URL":     "http://loja.example.com",
		"WP_APP_USER":     "mcp-content",
		"WP_APP_PASSWORD": "test-app-password-123",
	})
	// Act
	_, err := f.GetConfig()
	// Assert
	if !guard.IsCode(err, guard.AuthMissing) {
		t.Fatalf("got %v, want AUTH_MISSING", err)
	}
}

func TestGetConfigIncompleteAppPasswordIsAuthMissing(t *testing.T) {
	f := testFactory(map[string]string{
		"WP_BASE_URL":     "https://loja.example.com",
		"WP_APP_PASSWORD": "test-app-password-123",
	})
	_, err := f.GetConfig()
	if !guard.IsCode(err, guard.AuthMissing) {
		t.Fatalf("got %v, want AUTH_MISSING", err)
	}
}

func TestGetConfigDotEnvFileDoesNotOverrideProcessEnv(t *testing.T) {
	content := []byte("WP_BASE_URL=https://from-file.example.com\nWP_APP_USER=file-user\nWP_APP_PASSWORD=file-password-123\n")
	f := NewEnvConfigFactory(func(k string) string {
		switch k {
		case "DOTENV_PATH":
			return "/tmp/fake.env"
		case "WP_BASE_URL":
			return "https://loja.example.com"
		case "WP_APP_USER":
			return "mcp-content"
		case "WP_APP_PASSWORD":
			return "test-app-password-123"
		default:
			return ""
		}
	}).withReadFile(func(path string) ([]byte, error) {
		if path != "/tmp/fake.env" {
			return nil, errors.New("unexpected path")
		}
		return content, nil
	})
	cfg, err := f.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if cfg.BaseURL != "https://loja.example.com" {
		t.Fatalf("process env should win, got %q", cfg.BaseURL)
	}
}

func TestGetConfigErrorDoesNotContainPassword(t *testing.T) {
	secret := "super-secret-password-xyz"
	f := testFactory(map[string]string{
		"WP_APP_PASSWORD": secret,
	})
	_, err := f.GetConfig()
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("leaked secret in error: %v", err)
	}
}
