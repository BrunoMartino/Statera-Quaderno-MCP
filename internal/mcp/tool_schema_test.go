package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestListToolsOmitsOutputSchema(t *testing.T) {
	cs, _, _ := connectRuntime(t, http.NotFoundHandler())
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tools) == 0 {
		t.Fatal("no tools")
	}
	for _, tool := range res.Tools {
		if tool.OutputSchema != nil {
			t.Fatalf("%s has outputSchema", tool.Name)
		}
	}
}

func TestListPagesStructuredContentIsObject(t *testing.T) {
	cs, _, _ := connectRuntime(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":3,"slug":"sobre","status":"publish"}]`))
	}))
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "list_pages", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	if !json.Valid(raw) || raw[0] != '{' {
		t.Fatalf("structuredContent not object: %s", raw)
	}
}
