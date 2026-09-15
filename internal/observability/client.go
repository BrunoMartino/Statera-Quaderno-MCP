package observability

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"woocommerce-store-mcp/internal/guard"
)

const (
	debugPath  = "/wp-json/statera-mcp/v1/debug"
	logsPath   = "/wp-json/statera-mcp/v1/logs"
	secretHead = "X-Statera-MCP-Secret"
)

// Client is the adapter for the store companion plugin (statera-mcp/v1):
// debug-mode toggle and log collection. One instance per process, one store.
type Client struct {
	caller *guard.StoreCaller
	policy *guard.AccessPolicy
	secret string
	now    func() time.Time
}

func NewClient(caller *guard.StoreCaller, policy *guard.AccessPolicy, secret string) *Client {
	return &Client{caller: caller, policy: policy, secret: strings.TrimSpace(secret), now: time.Now}
}

func (c *Client) headers() (map[string]string, error) {
	if c.secret == "" {
		return nil, guard.NewError(guard.ObservabilitySecretMissing)
	}
	return map[string]string{secretHead: c.secret}, nil
}

// GetDebugMode reports whether WP_DEBUG / WP_DEBUG_LOG are on and where the log
// lives. Read-only: turning debug on or off is a host concern (WORDPRESS_DEBUG).
func (c *Client) GetDebugMode(ctx context.Context) (*DebugState, error) {
	h, err := c.headers()
	if err != nil {
		return nil, err
	}
	b, err := c.caller.DoWithHeaders(ctx, http.MethodGet, debugPath, nil, "", h)
	if err != nil {
		return nil, err
	}
	return decodeState(b)
}

// CollectLogs reads debug, WooCommerce, cron/schedule and callback entries.
func (c *Client) CollectLogs(ctx context.Context, q LogQuery) (*LogCollection, error) {
	source := strings.TrimSpace(strings.ToLower(q.Source))
	if source == "" {
		source = "all"
	}
	if err := c.policy.AssertLogSource(source); err != nil {
		return nil, err
	}
	level := strings.TrimSpace(strings.ToLower(q.Level))
	if err := c.policy.AssertLogLevel(level); err != nil {
		return nil, err
	}
	limit, err := c.policy.AssertLogLimit(q.Limit)
	if err != nil {
		return nil, err
	}
	since, err := c.normalizeSince(q.Since)
	if err != nil {
		return nil, err
	}
	h, err := c.headers()
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("source", source)
	query.Set("limit", strconv.Itoa(limit))
	if level != "" && level != "all" {
		query.Set("level", level)
	}
	if since != "" {
		query.Set("since", since)
	}
	if s := strings.TrimSpace(q.Search); s != "" {
		query.Set("search", s)
	}

	b, err := c.caller.DoWithHeaders(ctx, http.MethodGet, logsPath+"?"+query.Encode(), nil, "", h)
	if err != nil {
		return nil, err
	}
	var out LogCollection
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	entries := make([]LogEntry, 0, len(out.Entries))
	for _, e := range out.Entries {
		entries = append(entries, sanitizeEntry(e))
	}
	out.Entries = entries
	if out.Source == "" {
		out.Source = source
	}
	return &out, nil
}

// normalizeSince accepts RFC3339 or a relative window (30m, 2h, 7d) and always
// sends the store an absolute UTC instant.
func (c *Client) normalizeSince(raw string) (string, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return "", nil
	}
	if t, err := time.Parse(time.RFC3339, strings.ToUpper(raw)); err == nil {
		return t.UTC().Format(time.RFC3339), nil
	}
	unit := raw[len(raw)-1:]
	n, err := strconv.Atoi(strings.TrimSpace(raw[:len(raw)-1]))
	if err != nil || n <= 0 {
		return "", guard.NewError(guard.FieldForbidden)
	}
	var d time.Duration
	switch unit {
	case "m":
		d = time.Duration(n) * time.Minute
	case "h":
		d = time.Duration(n) * time.Hour
	case "d":
		d = time.Duration(n) * 24 * time.Hour
	default:
		return "", guard.NewError(guard.FieldForbidden)
	}
	return c.now().UTC().Add(-d).Format(time.RFC3339), nil
}

func decodeState(b []byte) (*DebugState, error) {
	var state DebugState
	if err := json.Unmarshal(b, &state); err != nil {
		return nil, err
	}
	state.Notice = SanitizeLine(state.Notice)
	return &state, nil
}
