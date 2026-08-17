package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"woocommerce-store-mcp/internal/guard"
)

func TestListOrdersToolStripsEmailAndPassesStatus(t *testing.T) {
	var gotPath, gotQuery string
	cs, _, _ := connectRuntime(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query().Get("status")
		if r.Method != http.MethodGet {
			t.Errorf("write %s", r.Method)
		}
		io.WriteString(w, `[{"id":727,"number":"727","status":"processing","total":"29.35","date_paid":"2017-03-22T16:28:08","refunds":[],"billing":{"email":"john.doe@example.com","phone":"555","address_1":"969 Market"}}]`)
	}))
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_orders",
		Arguments: map[string]any{"status": "processing"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("tool error: %s", toolText(res))
	}
	if gotPath != "/wp-json/wc/v3/orders" || gotQuery != "processing" {
		t.Fatalf("path=%s status=%s", gotPath, gotQuery)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	if strings.Contains(string(raw), "john.doe@example.com") || strings.Contains(string(raw), "969 Market") {
		t.Fatalf("PII in %s", raw)
	}
	if !strings.Contains(string(raw), `"id":727`) {
		t.Fatalf("missing order: %s", raw)
	}
}

func TestListPaymentsAndShipmentsTools(t *testing.T) {
	cs, _, _ := connectRuntime(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `[{"id":1,"status":"completed","payment_method":"bacs","date_paid":"2017-03-22T16:28:08","date_completed":"2017-03-23T10:00:00","meta_data":[]}]`)
	}))
	pay, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "list_payments", Arguments: map[string]any{}})
	if err != nil || pay.IsError {
		t.Fatalf("payments: %v %s", err, toolText(pay))
	}
	raw, _ := json.Marshal(pay.StructuredContent)
	if !strings.Contains(string(raw), `"confirmed":true`) {
		t.Fatalf("payments: %s", raw)
	}
	ship, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "list_shipments", Arguments: map[string]any{}})
	if err != nil || ship.IsError {
		t.Fatalf("shipments: %v %s", err, toolText(ship))
	}
	raw, _ = json.Marshal(ship.StructuredContent)
	if !strings.Contains(string(raw), `"fulfillment_state":"shipped"`) {
		t.Fatalf("shipments: %s", raw)
	}
}

func TestListOrdersAuthMissing(t *testing.T) {
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
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "list_orders", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError || !strings.Contains(toolText(res), guard.AuthMissing) {
		t.Fatalf("got %q", toolText(res))
	}
}
