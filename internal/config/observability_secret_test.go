package config

import "testing"

func TestObservabilitySecretFromEnv(t *testing.T) {
	env := map[string]string{
		"WP_BASE_URL":                 "https://loja.example.com",
		"WP_APP_USER":                 "mcp-content",
		"WP_APP_PASSWORD":             "aaaa bbbb cccc",
		"WP_MCP_OBSERVABILITY_SECRET": " obs-secret-123 ",
	}
	f := NewEnvConfigFactory(func(k string) string { return env[k] }).withReadFile(func(string) ([]byte, error) {
		return nil, errNoFile{}
	})
	cfg, err := f.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ObservabilitySecret != "obs-secret-123" {
		t.Fatalf("secret = %q", cfg.ObservabilitySecret)
	}
}

func TestBootSucceedsWithoutObservabilitySecret(t *testing.T) {
	env := map[string]string{
		"WP_BASE_URL":     "https://loja.example.com",
		"WP_APP_USER":     "mcp-content",
		"WP_APP_PASSWORD": "aaaa bbbb cccc",
	}
	f := NewEnvConfigFactory(func(k string) string { return env[k] }).withReadFile(func(string) ([]byte, error) {
		return nil, errNoFile{}
	})
	cfg, err := f.GetConfig()
	if err != nil {
		t.Fatalf("missing observability secret must not fail boot: %v", err)
	}
	if cfg.ObservabilitySecret != "" {
		t.Fatalf("secret = %q", cfg.ObservabilitySecret)
	}
}

type errNoFile struct{}

func (errNoFile) Error() string { return "no dotenv" }
