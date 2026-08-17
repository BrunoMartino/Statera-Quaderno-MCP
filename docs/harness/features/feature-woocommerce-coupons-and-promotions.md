# feature-woocommerce-coupons-and-promotions.md

## Feature: woocommerce-coupons-and-promotions

### Depende de

- feature-env-auth (auth para `/wc/v3/coupons`)
- feature-mcp-runtime (tools `list_coupons`, `create_coupon`, `update_coupon`, `delete_coupon`)

### Descrição

o usuario deve poder criar, editar e excluir cupons de desconto, assim como definir suas datas de de validade, os cupons devem poder ser setados como % ou valor absoluto, e se serão sobre o valor do carriho ou do tipo de produto atrelado a ele.

Promoção = o mesmo que cupom (não é `sale_price` em produto).

### Problema

vai mitigar a dificuldade de criar e manipular cupons no woocommerce, removendo ambiguidades que o painal admin atual causa

### Solução e trade-offs

deve ter as tools create, list, update e delete, seguindo o mesmo padrão que expliquei na pergunta 1, o principal trade-off é o usuario delegar essa criação completamente para a iA, caso isso aconteça sempre travave criações de cupons de mais de 20% e pergunte via AskQuestion se o usuário quer isso mesmo

Tools: `list_coupons`, `create_coupon`, `update_coupon`, `delete_coupon`. Paths: GET/POST `/wp-json/wc/v3/coupons`, GET/PATCH/DELETE `/wp-json/wc/v3/coupons/{id}`.

Allowlist de escrita: código; `discount_type` (`percent` | `fixed_cart` | `fixed_product`); `amount`; datas de validade; âmbito carrinho vs produto/categoria atrelado.

Gate: se `discount_type` é `percent` e `amount` > 20, a tool **não** chama a loja até o humano confirmar via AskQuestion. Sem confirmação → `DISCOUNT_CONFIRMATION_REQUIRED`.

### Fluxo (given/when/then)

- Dado env válido
- Quando `list_coupons` / `create_coupon` / `update_coupon` / `delete_coupon` com campos allowlisted e percentagem ≤ 20
- Então a tool chama `/wc/v3/coupons` com o método correspondente

- Dado `create_coupon` ou `update_coupon` com `discount_type=percent` e `amount` > 20
- Quando o agente tenta gravar sem confirmação humana nesta conversa
- Então não há HTTP; AskQuestion; só depois da confirmação a tool envia o body

### Casos de erro (explícitos, não deixar implícito)

- percentagem > 20 sem confirmação → `DISCOUNT_CONFIRMATION_REQUIRED`, zero HTTP
- chave fora da allowlist de cupom → `FIELD_FORBIDDEN`
- path que não seja `/wc/v3/coupons` → `ROUTE_FORBIDDEN`
- DELETE de post, página, produto ou media → `METHOD_FORBIDDEN` (DELETE só cupom)
- Auth em falta → `AUTH_MISSING`

### Critério de aceite (o que prova que está pronto)

- [ ] Teste: `create_coupon` percent 10 + validade + `fixed_cart` ou produto atrelado → POST `/wc/v3/coupons`
- [ ] Teste: `list_coupons` / `update_coupon` / `delete_coupon` usam só `/wc/v3/coupons` e `/{id}`
- [ ] Teste: `create_coupon` ou `update_coupon` percent > 20 sem confirmação → `DISCOUNT_CONFIRMATION_REQUIRED`, fake WC não recebe o body
- [ ] Teste: percent > 20 com confirmação humana → POST/PATCH segue
- [ ] Teste: DELETE cupom permitido; DELETE produto/post continua `METHOD_FORBIDDEN`
- [ ] Teste: `sale_price` / `regular_price` em produto continua `FIELD_FORBIDDEN`

### Exemplo / contexto

(pedido explícito: completar com o já respondido) painel admin Woo ambíguo vs tools `create`/`list`/`update`/`delete` com validade, `%` ou valor absoluto, âmbito carrinho vs tipo de produto; criação >20% pára e AskQuestion.

### Design Patterns (Gang of Four) Sugerido

- Adapter — `internal/woocommerce` para `/wc/v3/coupons`
- Policy — allowlist de campos + INV-006 excepção só cupom
- o gate 20% é confirmação (AskQuestion), não silêncio
