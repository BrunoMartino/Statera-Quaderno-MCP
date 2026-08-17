# forbidden_patterns.md

## Purpose

List patterns, architectures, and implementation habits that are forbidden by default.

## Forbidden Architectures By Default

The following are forbidden unless explicitly requested by the user or enabled in `architecture_rules.md`:

- Domain-Driven Design / DDD
- Clean Architecture
- Hexagonal Architecture
- Onion Architecture
- Ports and Adapters as a global architecture
- CQRS as a global architecture
- Event Sourcing as a global architecture

These patterns may be valid in other contexts, but they are not default choices for this project.

Also forbidden for this product:

- Browser page that calls `/wp-json` (CORS + cookies + XSS).
- Proxy genérico da WP REST ou da WooCommerce REST.
- Ligar agentes ao WooCommerce MCP nativo (`/woocommerce/mcp`).
- Multi-loja no mesmo processo (várias `WP_BASE_URL` no mesmo binário em execução).

## Forbidden MVC Violations

- Large controllers containing complex business workflows.
- Views that query databases directly.
- Models that call external APIs directly.
- Jobs that duplicate business logic instead of calling services.
- Services that depend directly on HTTP request/response objects.
- Controllers that contain authorization rules inline instead of using policies/guards.
- MCP tools that perform HTTP to WordPress/WooCommerce without `internal/guard`.
- Returning unsanitized WP/WC JSON as a tool result.

## Forbidden Abstraction Patterns

- Creating interfaces for every class by default.
- Creating repositories for every model by default.
- Creating factories for simple constructors.
- Creating service layers that only pass through to one method.
- Using “Manager”, “Helper”, or “Util” as vague dumping grounds.
- Adding patterns because they are fashionable rather than needed.
- Modeling WP/WC resources as `map[string]any` instead of structs.
- Generic tools: `wp_request`, `fetch`, `sql`, `wp_cli`, `eval`.

## Forbidden Data Patterns

- Updating critical state without transactions when consistency is required.
- Using nullable fields to encode many unrelated states.
- Persisting derived values without invalidation rules.
- Relying on timestamps alone for critical ordering.
- Mixing tenant/customer data without explicit scoping.
- Writing price, sale dates, stock, SKU, tax, shipping, downloads, `meta_data`, or other §6.2 forbidden keys.
- Creating products via MCP (POST `/wc/v3/products`).
- Adding POST, PUT, PATCH, or DELETE for orders, payments, shipping, customers, or settings. GET of those sections is allowed only when a feature harness allowlists the GET paths after an explicit human request.
- DELETE of any resource except `DELETE /wc/v3/coupons/{id}` (unpublish post/page = `status: draft` via PATCH).
- Writing product variations (`/wc/v3/products/{id}/variations`) in v1.
- Batch endpoints (`/batch`).
- Putting `consumer_key` / `consumer_secret` in query strings or logs.
- Accepting `base_url`, passwords, or keys in tool arguments.
- Committing `.env` or hardcoding store URL, user, or password.
- Alargar a allowlist pública REST da loja ou CORS para `*` / host do MCP.

## Forbidden Error Patterns

- Catching errors and returning success.
- Swallowing exceptions without logs or recovery.
- Retrying non-idempotent operations blindly.
- Logging secrets, tokens, passwords, or sensitive PII.
- Throwing generic errors for known business failures.
- Silent strip of forbidden keys then applying the rest (must be `FIELD_FORBIDDEN`).
- Retry on POST/PATCH to the store.
- Following HTTP redirects to a host other than `WP_BASE_URL`.

## Forbidden Agent Patterns

- Introducing DDD/Clean Architecture without permission.
- Expanding task scope beyond the user request.
- Performing broad refactors for small changes.
- Adding dependencies without justification.
- Changing architecture because a generic best practice suggests it.
- Adding MCP tools outside `my_docs/mcpContext.md` §7.
- Implementing the PHP store filters (context §8) inside this Go repository.
- Suggesting Consumer Keys of admin, disabling REST deny-by-default, or pasting passwords in chat.
- Publishing generated content without human confirmation when the tool receives `status` (default create: `draft`).
