package mcpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCreateCouponAmountAsNumberAtMCP(t *testing.T) {
	cs, _, _ := connectRuntime(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"id":9,"code":"N","discount_type":"percent","amount":"5"}`)
	}))
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "create_coupon",
		Arguments: map[string]any{
			"code":          "N",
			"discount_type": "percent",
			"amount":        5,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("tool error: %s", toolText(res))
	}
}

func TestCouponAmountSchemaAllowsNumber(t *testing.T) {
	cs, _, _ := connectRuntime(t, http.NotFoundHandler())
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range res.Tools {
		if tool.Name != "create_coupon" {
			continue
		}
		raw, _ := json.Marshal(tool.InputSchema)
		if !strings.Contains(string(raw), `"amount"`) {
			t.Fatalf("missing amount in %s", raw)
		}
		return
	}
	t.Fatal("create_coupon missing")
}
