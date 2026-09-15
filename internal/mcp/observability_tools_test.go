package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"woocommerce-store-mcp/internal/guard"
)

func TestCollectLogsToolStripsPII(t *testing.T) {
	cs, _, _ := connectRuntime(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wp-json/statera-mcp/v1/logs" || r.Method != http.MethodGet {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		io.WriteString(w, `{"source":"woocommerce","entries":[{"source":"woocommerce","channel":"fatal-errors","level":"critical","timestamp":"2026-09-15T10:00:00Z","message":"Order 727 falhou: cliente jane@example.com, tel +351 912 345 678"}],"truncated":false}`)
	}))

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "collect_logs",
		Arguments: map[string]any{"source": "woocommerce", "level": "critical", "limit": 50},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("tool error: %s", toolText(res))
	}
	raw, _ := json.Marshal(res.StructuredContent)
	for _, leak := range []string{"jane@example.com", "912 345 678"} {
		if bytes.Contains(raw, []byte(leak)) {
			t.Fatalf("leaked %q: %s", leak, raw)
		}
	}
	if !bytes.Contains(raw, []byte("fatal-errors")) {
		t.Fatalf("channel lost: %s", raw)
	}
}

// Não existe tool para ligar debug: isso é do host, não deste MCP.
func TestNoDebugToggleTool(t *testing.T) {
	cs, _, _ := connectRuntime(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
	}))
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range res.Tools {
		if tool.Name == "set_debug_mode" || tool.Name == "enable_debug" {
			t.Fatalf("tool de escrita de debug registada: %s", tool.Name)
		}
	}
	if _, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "set_debug_mode",
		Arguments: map[string]any{"enabled": true},
	}); err == nil {
		t.Fatal("set_debug_mode não devia existir")
	}
}

func TestDebugToolsRejectNonAllowlistedArguments(t *testing.T) {
	cs, _, _ := connectRuntime(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
	}))
	cases := []struct {
		tool string
		args map[string]any
	}{
		{"get_debug_mode", map[string]any{"enabled": true}},
		{"collect_logs", map[string]any{"source": "debug", "path": "../../wp-config.php"}},
		{"collect_logs", map[string]any{"source": "debug", "limit": 501}},
		{"collect_logs", map[string]any{"source": "wp-config"}},
	}
	for _, c := range cases {
		res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: c.tool, Arguments: c.args})
		if err != nil {
			t.Fatalf("%s: %v", c.tool, err)
		}
		if !res.IsError {
			t.Fatalf("%s %v: expected refusal", c.tool, c.args)
		}
	}
}

func TestObservabilityToolsWithoutSecretFailClosed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("store contacted without the observability secret: %s", r.URL.Path)
	}))
	t.Cleanup(ts.Close)

	cfg := testCfg()
	cfg.ObservabilitySecret = ""
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	policy := guard.NewAccessPolicy(nil)
	caller := guard.NewStoreCaller(ts.URL, testAuth{}, policy)
	rt := New(cfg, caller, logger)

	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := rt.Server().Connect(ctx, t1, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0"}, nil).Connect(ctx, t2, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, tool := range []string{"get_debug_mode", "collect_logs"} {
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: map[string]any{}})
		if err != nil {
			t.Fatalf("%s: %v", tool, err)
		}
		if !res.IsError || !strings.Contains(toolText(res), guard.ObservabilitySecretMissing) {
			t.Fatalf("%s: want OBSERVABILITY_SECRET_MISSING, got %s", tool, toolText(res))
		}
	}
}

func TestObservabilitySecretNeverLogged(t *testing.T) {
	cs, _, logBuf := connectRuntime(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"wp_debug":true,"wp_debug_log":true,"wp_debug_display":false}`)
	}))
	if _, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "get_debug_mode", Arguments: map[string]any{}}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(logBuf.String(), "obs-secret-123") {
		t.Fatalf("secret leaked to logs: %s", logBuf.String())
	}
}
