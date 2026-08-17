# Fase 1 — env-auth (Factory)

## Contexto

Slice: `internal/config` Factory a partir do env. Testes Red em `internal/config/factory_test.go`.

## Comando Red

```
go test ./internal/config -count=1
```

Falha esperada: package/símbolos inexistentes (`GetConfig`, `AUTH_MISSING`).

## Passos Green

1. Criar `internal/config` com `ConfigFactory` / `EnvConfigFactory.GetConfig`.
2. Produtos de auth: Application Password ou par WC consumer (Basic, nunca query).
3. Recusar env incompleto com `AUTH_MISSING`; `WP_BASE_URL` HTTPS sem trailing slash; um host.
4. `.env.example` com placeholders de `my_docs/mcpContext.md` §4.3. Não escrever `.env`.
5. Não logar nem incluir secrets em `error.Error()`.

Não implementar: tools MCP, HTTP à loja, PHP.

## Verificação

`go test ./internal/config -count=1` passa.
