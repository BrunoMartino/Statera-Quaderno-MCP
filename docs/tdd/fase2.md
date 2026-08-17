# Fase 2 — mcp-runtime + Policy (`internal/guard`, `internal/mcp`, `cmd/`)

## Contexto

Slice: Policy deny-by-default (rotas §5, métodos, campos) e servidor MCP (stdio + Streamable HTTP). Testes em `internal/guard/*_test.go` e `internal/mcp/*_test.go`.

## Comando Red

```
go test ./internal/guard ./internal/mcp -count=1
```

## Passos Green

1. `internal/guard`: `AccessPolicy`, códigos fixos, `assert_no_forbidden_keys`, gate percent > 20.
2. HTTP store: timeout 10s; GET ≤ 1 retry; POST/PATCH/DELETE 0 retry; recusar redirect off-host.
3. `internal/mcp`: conjunto fechado §7; instructions; `mcp.AddTool`; stdio e `NewStreamableHTTPHandler`.
4. `cmd/woocommerce-store-mcp`: boot Factory; stdio default; `-http` Streamable HTTP.

Não implementar ainda: bodies WP/WC completos (stubs ok até fase 3).

## Verificação

`go test ./internal/guard ./internal/mcp -count=1` passa.
