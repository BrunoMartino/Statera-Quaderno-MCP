package mcpserver

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"woocommerce-store-mcp/internal/config"
	"woocommerce-store-mcp/internal/guard"
)

type testAuth struct{}

func (testAuth) Apply(req *http.Request) {
	req.SetBasicAuth("mcp-content", "test-key-123")
}

func testCfg() *config.Config {
	return &config.Config{
		StoreID:         "minha-loja",
		Environment:     "staging",
		BaseURL:         "https://loja.example.com",
		AllowedStatuses: []string{"draft", "publish", "pending"},
		Auth:            testAuth{},
	}
}

func connectRuntime(t *testing.T, h http.Handler) (*mcp.ClientSession, *httptest.Server, *bytes.Buffer) {
	t.Helper()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	policy := guard.NewAccessPolicy(nil)
	caller := guard.NewStoreCaller(ts.URL, testAuth{}, policy)
	caller.Logger = logger
	caller.StoreID = "minha-loja"
	caller.Environment = "staging"
	rt := New(testCfg(), caller, logger)
	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()
	ss, err := rt.Server().Connect(ctx, t1, nil)
	if err != nil {
		t.Fatal(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0"}, nil)
	cs, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cs.Close()
		_ = ss.Wait()
	})
	return cs, ts, &logBuf
}

func TestListToolsIsClosedSetFromSection7(t *testing.T) {
	cs, _, _ := connectRuntime(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected HTTP %s %s", r.Method, r.URL.Path)
	}))
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(res.Tools))
	for _, tool := range res.Tools {
		got = append(got, tool.Name)
	}
	want := append([]string{}, closedToolNames...)
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("tools=%v want=%v", got, want)
	}
	if slices.Contains(got, "wp_request") || slices.Contains(got, "fetch") || slices.Contains(got, "sql") {
		t.Fatal("generic tool present")
	}
}

func TestInstructionsRefusePriceStockAndCouponGate(t *testing.T) {
	cs, _, _ := connectRuntime(t, http.NotFoundHandler())
	init := cs.InitializeResult()
	if init == nil {
		t.Fatal("missing initialize result")
	}
	text := init.Instructions
	for _, needle := range []string{"preço", "stock", "AskQuestion", "draft", "cupom"} {
		if !strings.Contains(strings.ToLower(text), strings.ToLower(needle)) && !strings.Contains(text, needle) {
			t.Fatalf("instructions missing %q: %s", needle, text)
		}
	}
}

func TestUpdateProductRegularPriceIsFieldForbidden(t *testing.T) {
	var hits int
	cs, _, _ := connectRuntime(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		io.WriteString(w, `{}`)
	}))
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "update_product_content",
		Arguments: map[string]any{"id": 5, "regular_price": "10"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || !res.IsError {
		t.Fatal("expected tool error")
	}
	if !strings.Contains(toolText(res), guard.FieldForbidden) {
		t.Fatalf("got %q", toolText(res))
	}
	if hits != 0 {
		t.Fatalf("HTTP hits=%d", hits)
	}
}

func TestUpsertPostRejectsBaseURLArg(t *testing.T) {
	var hits int
	cs, _, _ := connectRuntime(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
	}))
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "upsert_post",
		Arguments: map[string]any{"title": "x", "base_url": "https://evil.example"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError || !strings.Contains(toolText(res), guard.FieldForbidden) {
		t.Fatalf("got isError=%v text=%q", res.IsError, toolText(res))
	}
	if hits != 0 {
		t.Fatalf("hits=%d", hits)
	}
}

func TestCreateCouponOver20WithoutConfirm(t *testing.T) {
	var hits int
	cs, _, logBuf := connectRuntime(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
	}))
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "create_coupon",
		Arguments: map[string]any{
			"code":          "BIG",
			"discount_type": "percent",
			"amount":        "25",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError || !strings.Contains(toolText(res), guard.DiscountConfirmationRequired) {
		t.Fatalf("got %q", toolText(res))
	}
	if hits != 0 {
		t.Fatalf("hits=%d", hits)
	}
	if strings.Contains(logBuf.String(), "test-key-123") {
		t.Fatalf("secret in logs: %s", logBuf.String())
	}
}

func TestUpsertPostHappyPath(t *testing.T) {
	cs, _, _ := connectRuntime(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPost || r.URL.Path != "/wp-json/wp/v2/posts" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if !strings.Contains(string(body), `"status":"draft"`) {
			t.Errorf("body=%s", body)
		}
		w.Write([]byte(`{"id":3,"title":{"rendered":"Hi"},"status":"draft","type":"post"}`))
	}))
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "upsert_post",
		Arguments: map[string]any{"title": "Hi", "content": "Body"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("tool error: %s", toolText(res))
	}
}

func TestUploadMediaTool(t *testing.T) {
	cs, _, _ := connectRuntime(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wp-json/wp/v2/media" {
			t.Errorf("path %s", r.URL.Path)
		}
		w.Write([]byte(`{"id":77}`))
	}))
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "upload_media",
		Arguments: map[string]any{
			"filename":       "a.jpg",
			"content_base64": base64.StdEncoding.EncodeToString([]byte("img")),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("%s", toolText(res))
	}
	raw, _ := json.Marshal(res.StructuredContent)
	if !strings.Contains(string(raw), `"id":77`) && !strings.Contains(toolText(res), "77") {
		t.Fatalf("missing id in %s / %s", raw, toolText(res))
	}
}

func TestStreamableHTTPExposesSameTools(t *testing.T) {
	policy := guard.NewAccessPolicy(nil)
	caller := guard.NewStoreCaller("https://loja.example.com", testAuth{}, policy)
	rt := New(testCfg(), caller, slog.New(slog.NewTextHandler(io.Discard, nil)))
	hs := httptest.NewServer(rt.HTTPHandler())
	t.Cleanup(hs.Close)
	ctx := context.Background()
	client := mcp.NewClient(&mcp.Implementation{Name: "http-test", Version: "v0"}, nil)
	cs, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: hs.URL, DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	res, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tools) != len(closedToolNames) {
		t.Fatalf("got %d tools", len(res.Tools))
	}
}

func TestMissingAuthIsAuthMissing(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	caller := guard.NewStoreCaller("https://loja.example.com", nil, guard.NewAccessPolicy(nil))
	rt := New(testCfg(), caller, logger)
	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()
	ss, err := rt.Server().Connect(ctx, t1, nil)
	if err != nil {
		t.Fatal(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0"}, nil)
	cs, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cs.Close()
		_ = ss.Wait()
	})
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "list_posts", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError || !strings.Contains(toolText(res), guard.AuthMissing) {
		t.Fatalf("got %q", toolText(res))
	}
}

func toolText(res *mcp.CallToolResult) string {
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}
