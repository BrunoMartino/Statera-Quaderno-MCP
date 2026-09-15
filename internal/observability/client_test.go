package observability

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"woocommerce-store-mcp/internal/guard"
)

type testAuth struct{}

func (testAuth) Apply(req *http.Request) { req.SetBasicAuth("mcp-content", "test-key-123") }

func newTestClient(t *testing.T, secret string, h http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	policy := guard.NewAccessPolicy(nil)
	caller := guard.NewStoreCaller(ts.URL, testAuth{}, policy)
	c := NewClient(caller, policy, secret)
	c.now = func() time.Time { return time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC) }
	return c, ts
}

func TestGetDebugModeSendsSecretAndReadsOnly(t *testing.T) {
	var gotSecret, gotAuth, gotMethod string
	c, _ := newTestClient(t, "obs-secret-123", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSecret = r.Header.Get("X-Statera-MCP-Secret")
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		io.WriteString(w, `{"wp_debug":true,"wp_debug_log":true,"wp_debug_display":false,"log_path":"wp-content/statera-mcp-logs/debug.log","log_writable":true}`)
	}))

	state, err := c.GetDebugMode(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("method = %s (o estado de debug é só leitura)", gotMethod)
	}
	if gotSecret != "obs-secret-123" {
		t.Fatalf("secret header = %q", gotSecret)
	}
	if gotAuth == "" {
		t.Fatal("Application Password auth must still be applied")
	}
	if !state.WPDebug || !state.WPDebugLog || state.WPDebugDisplay {
		t.Fatalf("state = %+v", state)
	}
}

func TestObservabilityWithoutSecretNeverReachesTheStore(t *testing.T) {
	called := false
	c, _ := newTestClient(t, "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	ctx := context.Background()
	if _, err := c.GetDebugMode(ctx); !guard.IsCode(err, guard.ObservabilitySecretMissing) {
		t.Fatalf("get: want OBSERVABILITY_SECRET_MISSING, got %v", err)
	}
	if _, err := c.CollectLogs(ctx, LogQuery{Source: "debug"}); !guard.IsCode(err, guard.ObservabilitySecretMissing) {
		t.Fatalf("collect: want OBSERVABILITY_SECRET_MISSING, got %v", err)
	}
	if called {
		t.Fatal("store contacted without the observability secret")
	}
}

func TestCollectLogsQueryAndRelativeSince(t *testing.T) {
	var gotQuery, gotPath string
	c, _ := newTestClient(t, "s", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		io.WriteString(w, `{"source":"cron","entries":[{"source":"cron","channel":"woocommerce_scheduled_sales","level":"warning","next_run":"2026-09-15T11:00:00Z","schedule":"daily","status":"overdue","overdue":true,"message":"atrasado 60 min"}],"truncated":false}`)
	}))

	out, err := c.CollectLogs(context.Background(), LogQuery{Source: "cron", Level: "warning", Since: "2h", Limit: 25})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/wp-json/statera-mcp/v1/logs" {
		t.Fatalf("path = %s", gotPath)
	}
	for _, want := range []string{"source=cron", "level=warning", "limit=25", "since=2026-09-15T10%3A00%3A00Z"} {
		if !strings.Contains(gotQuery, want) {
			t.Fatalf("query %q missing %q", gotQuery, want)
		}
	}
	if len(out.Entries) != 1 || out.Entries[0].Overdue == nil || !*out.Entries[0].Overdue {
		t.Fatalf("cron entry lost its state: %+v", out.Entries)
	}
}

func TestCollectLogsRejectsUnknownSourceBeforeHTTP(t *testing.T) {
	called := false
	c, _ := newTestClient(t, "s", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	ctx := context.Background()
	if _, err := c.CollectLogs(ctx, LogQuery{Source: "wp-config"}); !guard.IsCode(err, guard.FieldForbidden) {
		t.Fatalf("want FIELD_FORBIDDEN, got %v", err)
	}
	if _, err := c.CollectLogs(ctx, LogQuery{Source: "debug", Limit: guard.LogsLimitMax + 1}); !guard.IsCode(err, guard.FieldForbidden) {
		t.Fatalf("limit: want FIELD_FORBIDDEN, got %v", err)
	}
	if _, err := c.CollectLogs(ctx, LogQuery{Source: "debug", Since: "ontem"}); !guard.IsCode(err, guard.FieldForbidden) {
		t.Fatalf("since: want FIELD_FORBIDDEN, got %v", err)
	}
	if called {
		t.Fatal("store contacted with a rejected query")
	}
}

func TestCollectLogsSanitizesEntries(t *testing.T) {
	c, _ := newTestClient(t, "s", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"source":"debug","entries":[{"source":"debug","level":"error","message":"Pagamento falhou para john.doe@example.com tel +351 912 345 678 ip 192.168.1.44 cartao 4111 1111 1111 1111 token=abcdef123456 Authorization: Basic Y2s6Y3M=","context":"/var/www/wp-content/plugins/pay/pay.php:120"}],"truncated":false}`)
	}))
	out, err := c.CollectLogs(context.Background(), LogQuery{Source: "debug"})
	if err != nil {
		t.Fatal(err)
	}
	msg := out.Entries[0].Message
	for _, leak := range []string{"john.doe@example.com", "912 345 678", "192.168.1.44", "4111 1111 1111 1111", "abcdef123456", "Y2s6Y3M="} {
		if strings.Contains(msg, leak) {
			t.Fatalf("leaked %q in %q", leak, msg)
		}
	}
	if !strings.Contains(msg, "Pagamento falhou") {
		t.Fatalf("sanitizer destroyed the message: %q", msg)
	}
	if out.Entries[0].Context != "/var/www/wp-content/plugins/pay/pay.php:120" {
		t.Fatalf("context should survive: %q", out.Entries[0].Context)
	}
}

func TestSanitizeLineCapsLength(t *testing.T) {
	long := strings.Repeat("x", maxMessageRunes+500)
	got := SanitizeLine(long)
	if len([]rune(got)) > maxMessageRunes+20 {
		t.Fatalf("line not capped: %d runes", len([]rune(got)))
	}
	if !strings.HasSuffix(got, "truncated]") {
		t.Fatalf("missing truncation marker: %q", got[len(got)-30:])
	}
}
