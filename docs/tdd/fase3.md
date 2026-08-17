# Fase 3 — wordpress-content, media-upload, product-content, coupons

## Contexto

Adapters `internal/wordpress` e `internal/woocommerce` + tools wired. Testes `*_test.go` nesses packages e integração MCP com httptest.

## Comando Red

```
go test ./internal/wordpress ./internal/woocommerce ./internal/mcp -count=1
```

## Passos Green

1. WordPress: list/get/upsert posts e pages; default draft; structs allowlisted; sem author/email.
2. Media: POST `/wp/v2/media`; GET `/{id}`; sem DELETE/settings.
3. Produtos: GET sanitizado; PATCH §6.2; recusar create/delete/variations/preço.
4. Cupons: list/create/update/delete; percent > 20 sem `human_confirmed` → `DISCOUNT_CONFIRMATION_REQUIRED`.

Não implementar: PHP da loja; limite de bytes de media (TBD).

## Verificação

`go test ./... -count=1` passa.
