# feature-woocommerce-product-content.md

## Feature: woocommerce-product-content

### Depende de

- feature-env-auth (auth para `/wc/v3/products`)
- feature-mcp-runtime (tools `list_products`, `get_product_content`, `update_product_content`)

### Descrição

Manipular a WooCommerce API, com auth correcta já fornecida no env, só para conteúdo editorial de produtos: nome, descrição, descrição curta, slug, imagem/galeria, taxonomia editorial, `catalog_visibility` (`visible` | `search` | `hidden`).

Tools: `list_products` (id, name, slug, status, permalink), `get_product_content`, `update_product_content` (PATCH campos §6.2).

Escrita em `/wc/v3/products/{id}/variations` **desligada** na v1.

### Problema

Chaves WC `read_write` conseguem mudar preço e stock. O MCP nativo do Woo expõe `products-update` comercial. Editores precisam de alterar textos/imagens de produtos **sem** poder alterar preço de produto, stock, SKU, envios, impostos ou criar/apagar produto. Cupons/promoções são outra feature (`feature-woocommerce-coupons-and-promotions`).

### Solução e trade-offs

Só GET lista/detalhe e PATCH `/wp-json/wc/v3/products` e `/{id}`. Allowlist de escrita §6.2. Proibido no PATCH (erro explícito se o payload original as continha, não aplica o resto): `regular_price`, `sale_price`, `price`, sale dates, `on_sale`, stock/SKU, tax, shipping, downloads, `meta_data`, etc.

Leitura: pode devolver nome/descrição/imagens. **Não** incluir preços, stock, SKU, `meta_data`, emails, IDs de cliente.

Criar produto novo: nunca via MCP. DELETE: nunca.

Trade-off: variações ficam de fora na v1 (o mesmo objecto mistura nome com preço/stock). A capability WP não chega; o guard neste repo é obrigatório (PHP na loja é fora deste repo).

### Fluxo (given/when/then)

- Dado um produto existente na loja
- Quando `update_product_content` com `name` / `description` / `short_description` / `slug` / `images`
- Então PATCH só esses campos e a resposta da tool não tem preço/stock/SKU/`meta_data`

- Dado `list_products` / `get_product_content`
- Quando a WC REST devolve um produto completo
- Então a tool sanitiza antes de devolver ao LLM

### Casos de erro (explícitos, não deixar implícito)

- PATCH com `regular_price` (ou qualquer chave §6.2 proibida) → `FIELD_FORBIDDEN`
- POST criar produto ou DELETE → `METHOD_FORBIDDEN`
- Escrita em variations → `ROUTE_FORBIDDEN` / não há tool
- Pedido humano para mudar preço ou stock de produto, SKU → recusar. Cupons/promoções → `feature-woocommerce-coupons-and-promotions`.

### Critério de aceite (o que prova que está pronto)

- [ ] Teste: PATCH `description` → 200 no fake, body sem campos comerciais
- [ ] Teste: PATCH com `regular_price` → `FIELD_FORBIDDEN`, fake não recebe o campo
- [ ] Teste: `get_product_content` / `list_products` sem preço, stock, SKU, `meta_data`
- [ ] Teste: POST `/wc/v3/products` e DELETE nunca são emitidos
- [ ] Teste: nenhuma chamada a `/variations` em escrita

### Exemplo / contexto

Pedido original: manipular a WooCommerce API dado que as auth correctas tenham sido fornecidas. Detalhe em `my_docs/mcpContext.md` §6.2, §6.3, tools §7.

Motivo v1 sem variações: o mesmo objecto mistura nome da variação (conteúdo) com `regular_price` / `stock_quantity`.

### Design Patterns (Gang of Four) Sugerido

- Adapter — `internal/woocommerce` para GET/PATCH de produto editorial
- Policy — allowlist §6.2 + sanitização de resposta (INV-002, INV-008)
