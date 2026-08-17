package config

import (
	"net/url"
	"os"
	"strings"

	"woocommerce-store-mcp/internal/guard"
)

// ConfigFactory creates a Config from the process environment (Factory Method).
type ConfigFactory interface {
	GetConfig() (*Config, error)
}

// EnvConfigFactory is the concrete creator: env / DOTENV_PATH → Config or AUTH_MISSING.
type EnvConfigFactory struct {
	lookup   func(string) string
	readFile func(string) ([]byte, error)
}

// NewEnvConfigFactory returns a factory that reads process env and optional dotenv.
func NewEnvConfigFactory(lookup func(string) string) *EnvConfigFactory {
	if lookup == nil {
		lookup = os.Getenv
	}
	return &EnvConfigFactory{lookup: lookup, readFile: os.ReadFile}
}

func (f *EnvConfigFactory) withReadFile(readFile func(string) ([]byte, error)) *EnvConfigFactory {
	f.readFile = readFile
	return f
}

// GetConfig is the factory method: Application Password, else WC consumer, else AUTH_MISSING.
func (f EnvConfigFactory) GetConfig() (*Config, error) {
	fileVals := f.loadDotEnv()
	get := func(key string) string {
		if v := strings.TrimSpace(f.lookup(key)); v != "" {
			return v
		}
		return strings.TrimSpace(fileVals[key])
	}

	baseURL, err := normalizeBaseURL(get("WP_BASE_URL"))
	if err != nil {
		return nil, err
	}

	appUser := get("WP_APP_USER")
	appPassword := get("WP_APP_PASSWORD")
	wcKey := get("WC_CONSUMER_KEY")
	wcSecret := get("WC_CONSUMER_SECRET")

	var auth Auth
	switch {
	case appPassword != "":
		if appUser == "" {
			return nil, guard.NewError(guard.AuthMissing)
		}
		auth = applicationPasswordAuth{user: appUser, password: appPassword}
	case wcKey != "" || wcSecret != "":
		if wcKey == "" || wcSecret == "" {
			return nil, guard.NewError(guard.AuthMissing)
		}
		auth = wooCommerceConsumerAuth{key: wcKey, secret: wcSecret}
	default:
		return nil, guard.NewError(guard.AuthMissing)
	}

	statuses := parseStatuses(get("WP_MCP_ALLOWED_STATUSES"))
	userLogin := get("WP_MCP_USER_LOGIN")
	if userLogin == "" {
		userLogin = appUser
	}

	return &Config{
		StoreID:         get("WP_STORE_ID"),
		Environment:     get("WP_ENVIRONMENT"),
		BaseURL:         baseURL,
		UserLogin:       userLogin,
		AllowedStatuses: statuses,
		Auth:            auth,
	}, nil
}

func (f EnvConfigFactory) loadDotEnv() map[string]string {
	if f.readFile == nil {
		return nil
	}
	path := strings.TrimSpace(f.lookup("DOTENV_PATH"))
	if path == "" {
		path = ".env"
	}
	data, err := f.readFile(path)
	if err != nil {
		return nil
	}
	return parseDotEnv(data)
}

func normalizeBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", guard.NewError(guard.AuthMissing)
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return "", guard.NewError(guard.AuthMissing)
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func parseStatuses(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{"draft", "publish", "pending"}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"draft", "publish", "pending"}
	}
	return out
}
