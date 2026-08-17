# domain_invariantes.md

## Purpose

Define the business truths that must always remain valid.

## What Is A Domain Invariant?

A domain invariant is a rule that must always be true for the business, regardless of which controller, service, job, command, import, migration, or agent changed the system.

It is stronger than a simple validation.

A validation usually checks whether one input is acceptable.  
An invariant protects the consistency of the business state over time.

Example:

- Validation: “email must have a valid format.”
- Invariant: “a user cannot belong to two tenants with conflicting access scopes.”
- Validation: “order quantity must be greater than zero.”
- Invariant: “a paid order cannot be changed in a way that alters the charged amount without issuing an adjustment.”

## Invariant Template

### INV-001: One process, one store

Description:

- A running MCP process has exactly one `WP_BASE_URL`. It never mixes loja A with loja B, or staging with production, in the same process.

Business reason:

- Credentials and content belong to one site. Mixing hosts would edit the wrong store.

Must always be true when:

- Boot, every tool call, every HTTP request, redirects.

Applies to:

- Controllers: `internal/mcp`
- Models: config structs
- Services: all
- Jobs: N/A
- Events: N/A
- Database tables: N/A

Enforced by:

- Model validation: config load
- Service rule: no
- Database constraint: no
- Transaction: no
- Policy / Guard: refuse redirects to another host
- Test coverage: yes

Failure impact:

- Writes land on the wrong WordPress site.

### INV-002: Product price and stock never written; coupons allowed

Description:

- This MCP never changes product `regular_price`, `sale_price`, `price`, sale dates, `on_sale`, stock, SKU, orders, payments, shipping, users, accounts, settings, plugins, or themes.
- Promoções e cupons de desconto **são** permitidos só via `/wc/v3/coupons` (`feature-woocommerce-coupons-and-promotions`). Não é `sale_price` em produto.

Business reason:

- Copy/images stay editorial. Desconto de loja faz-se com cupom, não alterando o preço do produto. Stock e contas continuam fora.

Must always be true when:

- Any write tool, including “just to test” or “the admin asked in chat”.

Applies to:

- Controllers: write tools
- Models: product write structs (no price/stock); coupon write structs (allowlist de cupom)
- Services: `UpdateProductContent`, coupon create/update/delete
- Jobs: N/A
- Events: N/A
- Database tables: N/A

Enforced by:

- Model validation: allowlisted structs
- Service rule: yes
- Database constraint: no
- Transaction: no
- Policy / Guard: `assert_no_forbidden_keys` → `FIELD_FORBIDDEN`
- Test coverage: yes (PATCH `regular_price` fails; coupon POST allowed)

Failure impact:

- Store product money or stock changed by an agent, or discounts applied via product sale price.

Store-side PHP filters (context §8) are required on each shop but are **not** implemented in this repository. This repo must still enforce INV-002 in `internal/guard`.

### INV-003: Auth only from environment

Description:

- Store URL and credentials come from process env / `.env` (`DOTENV_PATH`). Never from git, source, tool arguments, or logs. Boot or tool fails with `AUTH_MISSING` if Application Password **or** the WC consumer pair is incomplete.

Business reason:

- The binary is reusable. Secrets stay per instance.

Must always be true when:

- Boot and every outbound HTTP call.

Applies to:

- Controllers: tools must not accept `base_url` / passwords
- Models: config
- Services: all
- Jobs: N/A
- Events: N/A
- Database tables: N/A

Enforced by:

- Model validation: required env
- Service rule: no
- Database constraint: no
- Transaction: no
- Policy / Guard: `AUTH_MISSING`
- Test coverage: yes

Failure impact:

- Credential leak or unauthenticated calls.

### INV-004: REST surface closed

Description:

- Outbound HTTP is only the paths and methods in `my_docs/mcpContext.md` §5. Anything else is `ROUTE_FORBIDDEN` or `METHOD_FORBIDDEN` with no retry. No generic WP/WC proxy. No `/woocommerce/mcp`.

Business reason:

- Open REST would expose users, orders, settings, and checkout.

Must always be true when:

- Every adapter request.

Applies to:

- Controllers: no `wp_request` tool
- Models: N/A
- Services: all
- Jobs: N/A
- Events: N/A
- Database tables: N/A

Enforced by:

- Model validation: no
- Service rule: no
- Database constraint: no
- Transaction: no
- Policy / Guard: path/method allowlist
- Test coverage: yes (users GET never sent; DELETE except coupons forbidden)

Failure impact:

- Agent reaches accounts, orders, or settings.

### INV-005: Forbidden keys fail the tool

Description:

- If a write payload contains a forbidden or unknown key, the tool returns `FIELD_FORBIDDEN` and does not apply the rest of the body.

Business reason:

- Silent strip would let the model believe price or stock changed.

Must always be true when:

- POST/PATCH bodies for posts, pages, products, coupons, media metadata.

Applies to:

- Controllers: write tools
- Models: write structs
- Services: upserts and product update
- Jobs: N/A
- Events: N/A
- Database tables: N/A

Enforced by:

- Model validation: yes
- Service rule: yes
- Database constraint: no
- Transaction: no
- Policy / Guard: `assert_no_forbidden_keys`
- Test coverage: yes

Failure impact:

- Partial writes and false success.

### INV-006: No DELETE except coupons

Description:

- The MCP never sends DELETE except `DELETE /wc/v3/coupons/{id}`. Unpublish of posts/pages is `status: draft` via PATCH. Products and media are never deleted.

Business reason:

- Content removal is not an editor action. Excluir cupom é parte da feature de promoções.

Must always be true when:

- Any tool.

Applies to:

- Controllers: `delete_coupon` only
- Models: N/A
- Services: all
- Jobs: N/A
- Events: N/A
- Database tables: N/A

Enforced by:

- Model validation: no
- Service rule: yes
- Database constraint: no
- Transaction: no
- Policy / Guard: `METHOD_FORBIDDEN` except coupon DELETE
- Test coverage: yes

Failure impact:

- Irreversible content loss on the store.

### INV-007: No product create via MCP

Description:

- The MCP never POST-creates a WooCommerce product. Product writes are PATCH of editorial fields on an existing id.

Business reason:

- Creating a product implies price and stock.

Must always be true when:

- Product tools.

Applies to:

- Controllers: no `create_product`
- Models: no product-create body
- Services: product content only
- Jobs: N/A
- Events: N/A
- Database tables: N/A

Enforced by:

- Model validation: no
- Service rule: yes
- Database constraint: no
- Transaction: no
- Policy / Guard: `METHOD_FORBIDDEN` on POST `/wc/v3/products`
- Test coverage: yes

Failure impact:

- New SKUs/prices created by an agent.

### INV-008: Responses omit commercial and PII fields

Description:

- Tool results may include editorial fields. Product responses must not include prices, stock, SKU, `meta_data`, emails, or detailed author. Coupon tools **may** return `code`, `discount_type`, `amount`, validade e âmbito (carrinho/produto).

Business reason:

- The model should not see or rewrite product money, inventory, or personal data. Cupom precisa do valor do desconto para a feature funcionar.

Must always be true when:

- Every tool response, especially `get_product_content` / `list_products`. Coupon list/get/create/update responses include coupon fields only.

Applies to:

- Controllers: tool results
- Models: response structs
- Services: all reads
- Jobs: N/A
- Events: N/A
- Database tables: N/A

Enforced by:

- Model validation: response types
- Service rule: sanitization before return
- Database constraint: no
- Transaction: no
- Policy / Guard: response strip
- Test coverage: yes

Failure impact:

- Leak of PII or commercial data into the agent context.

### INV-009: Percent coupon over 20 requires human confirmation

Description:

- `create_coupon` / `update_coupon` with `discount_type=percent` and `amount` > 20 must not call the store until the human confirms via AskQuestion in this conversation. Without confirmation the tool returns `DISCOUNT_CONFIRMATION_REQUIRED`.

Business reason:

- Delegating coupon creation to the agent can over-discount. The trade-off (user Q3) is a hard stop above 20%.

Must always be true when:

- Coupon create and update.

Applies to:

- Controllers: `create_coupon`, `update_coupon`
- Models: coupon amount/type
- Services: coupon write
- Jobs: N/A
- Events: N/A
- Database tables: N/A

Enforced by:

- Model validation: yes
- Service rule: yes
- Database constraint: no
- Transaction: no
- Policy / Guard: yes
- Test coverage: yes

Failure impact:

- Unconfirmed deep discounts applied to the store.

## Common MVC Enforcement Points

Model:

- Use for simple validations, relationships, and local consistency.

Service:

- Use for multi-step business workflows.

Policy / Guard:

- Use for authorization invariants.

Database:

- Use for uniqueness, foreign keys, non-null constraints, and critical consistency.
- Unused in this process.

Job:

- Use cautiously; jobs should preserve invariants, not redefine them.
- Unused.

Controller:

- Should not be the only place enforcing important invariants.

## Concurrency Rules

- Define which invariants require transactions: none in this process (store is the source of truth).
- Define which invariants require locking: none here.
- Define which invariants require idempotency keys: none; POST/PATCH are not retried by the client.
- Define which invariants can be eventually consistent: WP `modified` after write is whatever the store returns.

## Agent Rules

- Agents must check this file before changing business behavior.
- Agents must not weaken invariants without explicit approval.
- Any change affecting an invariant must include tests.
- Reviewer agents must identify touched invariants.
