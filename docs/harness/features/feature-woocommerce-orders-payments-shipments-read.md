# feature-woocommerce-orders-payments-shipments-read.md

## Feature: woocommerce-orders-payments-shipments-read

### Depende de

- feature-env-auth (auth para GET `/wc/v3/orders`)
- feature-mcp-runtime (tools `list_orders`, `list_payments`, `list_shipments`)

### Descrição

Três tools só GET: list_orders (filtro por status, p.ex. pagos, e se foi solicitado reembolso), list_payments (método, detalhes, confirmado ou não), list_shipments (código de rastreio, enviado vs aguardando). Sem POST/PATCH/DELETE.

### Problema

Hoje o MCP não deixa consultar pedidos, pagamentos nem envios; só conteúdo editorial e cupons.

### Solução e trade-offs

Só list/GET nas três tools. Trade-off: o agente passa a ver dados de encomenda/pagamento (PII comercial); escrita de pedidos/pagamentos/envios continua proibida. Respostas sanitizadas (sem email/telefone/morada completa), a menos que eu diga o contrário no Other.

### Fluxo (given/when/then)

- Dado env válido
- Quando `list_orders` / `list_payments` / `list_shipments` com filtros allowlisted
- Então a tool chama só GET `/wc/v3/orders` (e GET `/wc/v3/refunds` se o array `refunds` do pedido não chegar)

- Dado um pedido de POST, PUT, PATCH ou DELETE em orders/refunds/shipments
- Quando o agente tenta a operação
- Então `METHOD_FORBIDDEN` e zero HTTP de escrita

### Casos de erro (explícitos, não deixar implícito)

- POST/PUT/PATCH/DELETE em `/wc/v3/orders` ou `/wc/v3/refunds` → `METHOD_FORBIDDEN`
- path de users, settings, payment_gateways, shipping zones, `/batch` → `ROUTE_FORBIDDEN`
- Auth em falta → `AUTH_MISSING`
- Humano pede criar/alterar/apagar encomenda, pagamento ou envio → recusar

### Critério de aceite (o que prova que está pronto)

- [ ] Teste: `list_orders` com `status` → GET `/wc/v3/orders`; resposta inclui flag/array de reembolso; sem email/telefone/morada
- [ ] Teste: `list_payments` → projecção do pedido (`payment_method`, `date_paid`, `confirmed`); sem POST
- [ ] Teste: `list_shipments` → estado aguardando/enviado e rastreio só de meta allowlisted
- [ ] Teste: POST/PUT/PATCH/DELETE orders → `METHOD_FORBIDDEN`; fake WC não recebe escrita
- [ ] Teste: GET `/wc/v3/payment_gateways` e `/customers` nunca são emitidos

### Exemplo / contexto

Ex.: listar pedidos pagos; ver se foi solicitado reembolso; ver se o pagamento foi confirmado; ver se o envio já tem código de rastreio ou está aguardando.

### Design Patterns (Gang of Four) Sugerido

- Adapter — `internal/woocommerce` para GET `/wc/v3/orders`
- Policy — allowlist GET-only; sanitização sem email/telefone/morada; POST/PUT/PATCH/DELETE de pedidos `METHOD_FORBIDDEN`
