# Testes

| Suite / name | Purpose | Path | Isolated run |
|--------------|---------|------|----------------|
| env-auth Factory | AUTH_MISSING, App Password vs WC Basic, no consumer query, no secret leak | `internal/config/factory_test.go` | `go test ./internal/config -count=1` |
| AccessPolicy | routes §5, DELETE coupon vs post, FIELD_FORBIDDEN, percent >20 | `internal/guard/policy_test.go` | `go test ./internal/guard -count=1` |
| StoreCaller | GET retry once, POST no retry, off-host redirect, auth header, no log secret | `internal/guard/http_test.go` | `go test ./internal/guard -count=1` |
| WordPress client | upsert draft, password forbidden, get without email/author, media upload | `internal/wordpress/client_test.go` | `go test ./internal/wordpress -count=1` |
| WooCommerce client | PATCH description, regular_price forbidden, sanitise GET, coupons + 20% gate | `internal/woocommerce/client_test.go` | `go test ./internal/woocommerce -count=1` |
| WooCommerce list pagination | `list_products` percorre todas as páginas (`per_page=100`) | `internal/woocommerce/list_pagination_test.go` | `go test ./internal/woocommerce -count=1 -run TestListProductsWalksAllPages` |
| MCP runtime | closed tool set §7, instructions, FIELD_FORBIDDEN, AUTH_MISSING, stdio+HTTP | `internal/mcp/server_test.go` | `go test ./internal/mcp -count=1` |
| MCP tool schema | tools/list sem outputSchema (Cursor) | `internal/mcp/tool_schema_test.go` | `go test ./internal/mcp -count=1 -run TestListToolsOmitsOutputSchema` |
| cmd boot | boot without credentials → AUTH_MISSING | `cmd/woocommerce-store-mcp/main_test.go` | `go test ./cmd/woocommerce-store-mcp -count=1` |
