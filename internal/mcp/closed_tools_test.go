package mcpserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestAllClosedToolsHappyPathStructuredObject(t *testing.T) {
	cs, _, _ := connectRuntime(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/wp-json/wp/v2/posts":
			if r.Method == http.MethodGet {
				io.WriteString(w, `[{"id":1,"title":{"rendered":"Hi"},"slug":"hi","status":"publish"}]`)
				return
			}
			io.WriteString(w, `{"id":1,"title":{"raw":"Hi","rendered":"Hi"},"content":{"raw":"<p>x</p>","rendered":"<p>x</p>"},"status":"draft","type":"post"}`)
		case strings.HasPrefix(r.URL.Path, "/wp-json/wp/v2/posts/"):
			io.WriteString(w, `{"id":1,"title":{"raw":"Hi","rendered":"Hi"},"content":{"raw":"<p>x</p>","rendered":"<p>x</p>"},"status":"draft","type":"post"}`)
		case r.URL.Path == "/wp-json/wp/v2/pages":
			if r.Method == http.MethodGet {
				io.WriteString(w, `[{"id":2,"title":{"rendered":"P"},"slug":"p","status":"publish"}]`)
				return
			}
			io.WriteString(w, `{"id":2,"title":{"raw":"P","rendered":"P"},"content":{"raw":"<p>p</p>"},"status":"draft","type":"page"}`)
		case strings.HasPrefix(r.URL.Path, "/wp-json/wp/v2/pages/"):
			io.WriteString(w, `{"id":2,"title":{"raw":"P","rendered":"P"},"content":{"raw":"<p>p</p>"},"status":"draft","type":"page"}`)
		case r.URL.Path == "/wp-json/wc/v3/products":
			if r.Method == http.MethodGet {
				io.WriteString(w, `[{"id":5,"name":"Mug","slug":"mug","status":"publish"}]`)
				return
			}
			io.WriteString(w, `{"id":5,"name":"Mug","description":"D","short_description":"S"}`)
		case strings.HasPrefix(r.URL.Path, "/wp-json/wc/v3/products/"):
			io.WriteString(w, `{"id":5,"name":"Mug","description":"D","short_description":"S"}`)
		case r.URL.Path == "/wp-json/wc/v3/coupons":
			if r.Method == http.MethodGet {
				io.WriteString(w, `[{"id":8,"code":"SAVE10","discount_type":"percent","amount":"10"}]`)
				return
			}
			io.WriteString(w, `{"id":8,"code":"SAVE10","discount_type":"percent","amount":"10"}`)
		case strings.HasPrefix(r.URL.Path, "/wp-json/wc/v3/coupons/"):
			if r.Method == http.MethodDelete {
				io.WriteString(w, `{"id":8,"code":"SAVE10"}`)
				return
			}
			io.WriteString(w, `{"id":8,"code":"SAVE10","discount_type":"percent","amount":"10"}`)
		case r.URL.Path == "/wp-json/wc/v3/orders":
			io.WriteString(w, `[{"id":727,"number":"727","status":"processing","total":"29.35","date_paid":"2017-03-22T16:28:08","refunds":[]}]`)
		case r.URL.Path == "/wp-json/wp/v2/media":
			io.WriteString(w, `{"id":44,"mime_type":"image/png"}`)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	calls := []struct {
		name string
		args map[string]any
	}{
		{"list_posts", map[string]any{}},
		{"get_post", map[string]any{"id": 1}},
		{"upsert_post", map[string]any{"title": "Hi", "content": "Body"}},
		{"list_pages", map[string]any{}},
		{"get_page", map[string]any{"id": 2}},
		{"upsert_page", map[string]any{"title": "P", "content": "Body"}},
		{"list_products", map[string]any{}},
		{"get_product_content", map[string]any{"id": 5}},
		{"update_product_content", map[string]any{"id": 5, "name": "Mug"}},
		{"upload_media", map[string]any{"filename": "a.png", "content_base64": base64.StdEncoding.EncodeToString([]byte("img"))}},
		{"list_coupons", map[string]any{}},
		{"create_coupon", map[string]any{"code": "SAVE10", "discount_type": "percent", "amount": "10"}},
		{"update_coupon", map[string]any{"id": 8, "amount": "10"}},
		{"delete_coupon", map[string]any{"id": 8}},
		{"list_orders", map[string]any{"status": "processing"}},
		{"list_payments", map[string]any{}},
		{"list_shipments", map[string]any{}},
	}
	if len(calls) != len(closedToolNames) {
		t.Fatalf("calls=%d tools=%d", len(calls), len(closedToolNames))
	}
	for _, call := range calls {
		res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: call.name, Arguments: call.args})
		if err != nil {
			t.Fatalf("%s: %v", call.name, err)
		}
		if res.IsError {
			t.Fatalf("%s tool error: %s", call.name, toolText(res))
		}
		raw, _ := json.Marshal(res.StructuredContent)
		if !json.Valid(raw) || raw[0] != '{' {
			t.Fatalf("%s structuredContent not object: %s", call.name, raw)
		}
	}
}
