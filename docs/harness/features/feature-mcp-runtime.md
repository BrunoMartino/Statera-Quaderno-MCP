# feature-mcp-runtime.md

## Feature: mcp-runtime

### Depende de

- feature-env-auth (boot e credenciais antes de servir tools)

### Descrição

Servidor MCP **reutilizável** para qualquer loja WordPress + WooCommerce. Agentes (Cursor, Claude, etc.) editam conteúdo: páginas, posts, textos/imagens de produtos, e cupons/promoções (`/wc/v3/coupons`). Também listam pedidos, pagamentos e envios só via GET (`list_orders`, `list_payments`, `list_shipments`).

Mesmo binário MCP: carrega env da loja/ambiente, allowlist de tools + strip de campos proibidos, HTTPS para `WP_BASE_URL`.

Só estas tools existem. Não há `wp_request`, `fetch`, `sql`, `wp_cli`, `eval`.

Instruções do servidor e description de cada tool: páginas, posts, conteúdo de produto e cupons; listagens GET de pedidos/pagamentos/envios; recusar preço/stock de produto, contas, e **escrita** de encomenda; percentagem de cupom > 20 exige AskQuestion; não inventar tools; não pedir passwords no chat; default de criação de post/página `draft`.

Transporte: stdio (Cursor) e Streamable HTTP (Coolify), SDK oficial `modelcontextprotocol/go-sdk`.

### Problema

O WooCommerce traz um adapter MCP (`/woocommerce/mcp`) que expõe abilities REST, incluindo `woocommerce/products-update` (preço e stock), create/delete de produtos, encomendas e outras operações de loja.

Quem usa um MCP aberto consegue alterar dinheiro, stock, contas e operações da loja. Este caso de uso precisa de tools fechadas.

### Solução e trade-offs

Este projecto usa um MCP **próprio**, com tools fechadas (tabela §7). Feature `mcp_integration` do Woo deve permanecer desligada no site (fora deste repo).

Regras de cada tool de escrita: (1) JSON Schema só allowlisted (2) `assert_no_forbidden_keys` — erro, não strip silencioso (3) chamar WP (4) sanitizar a resposta.

Códigos fixos: `FIELD_FORBIDDEN`, `ROUTE_FORBIDDEN`, `METHOD_FORBIDDEN`, `AUTH_MISSING`, `DISCOUNT_CONFIRMATION_REQUIRED`.

Trade-off: não é proxy genérico da WP/WC REST; tools em falta ficam em falta. Não implementar o MCP como página no browser.

### Fluxo (given/when/then)

- Dado processo com env válido
- Quando o cliente MCP lista tools
- Então só existem as tools da tabela §7, com instruções de conteúdo + cupons

- Dado um pedido de escrita com chave proibida
- Quando a tool corre
- Então responde `FIELD_FORBIDDEN` e não chama a loja com esse body

### Casos de erro (explícitos, não deixar implícito)

- Tool ou path fora da tabela → não existe / `ROUTE_FORBIDDEN`
- DELETE de post/produto ou POST de produto → `METHOD_FORBIDDEN`
- Env incompleto → `AUTH_MISSING`
- Humano pede **alterar** preço/stock de produto, encomenda, cliente, plugin, setting → recusar. Listar encomendas/pagamentos/envios: tools GET. Cupons: tools de cupom; percent > 20 → AskQuestion
- POST/PUT/PATCH/DELETE em orders → `METHOD_FORBIDDEN`

### Critério de aceite (o que prova que está pronto)

- [ ] Teste: o servidor expõe exactamente as tools §7 (list/get/upsert posts e pages; list/get/update product content; upload_media; list/create/update/delete coupons; list_orders, list_payments, list_shipments)
- [ ] Teste: não existe tool de HTTP genérico
- [ ] Teste: stdio sobe com o mesmo binário que Streamable HTTP
- [ ] Teste: instructions/descriptions incluem recusa de preço/stock de produto, contas, e escrita de encomenda; cupom >20% AskQuestion; default `draft` em post/página
- [ ] Teste: códigos de erro fixos nas falhas de allowlist/auth/confirmação

### Exemplo / contexto

Arquitectura em `my_docs/mcpContext.md` §3:

```
Agente (Cursor / Claude)
    → mesmo binário MCP
        → carrega .env da loja/ambiente
        → allowlist de tools + strip de campos proibidos
        → HTTPS para WP_BASE_URL
```

Tools: `list_posts`, `get_post`, `upsert_post`, `list_pages`, `get_page`, `upsert_page`, `list_products`, `get_product_content`, `update_product_content`, `upload_media`, `list_coupons`, `create_coupon`, `update_coupon`, `delete_coupon`, `list_orders`, `list_payments`, `list_shipments`.

### Design Patterns (Gang of Four) Sugerido

- Adapter — encapsular o SDK MCP (stdio / Streamable HTTP) atrás de `internal/mcp`
- Policy — allowlist de tools, rotas, métodos e campos antes de qualquer I/O
- Command — cada tool é uma acção explícita, sem HTTP genérico
