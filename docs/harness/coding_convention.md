# coding_convention.md

## Purpose

Define coding style and implementation preferences for an MVC-based project.

## General Style

- Prefer clear Go in the package layout from `architecture_rules.md` (MVC roles mapped to MCP tools, services, adapters, structs, guard).
- Keep code readable before clever.
- Avoid premature abstraction.
- Follow existing project conventions.
- Use explicit names for business concepts (`Post`, `Page`, `ProductContent`, `FIELD_FORBIDDEN`).
- Keep functions and methods focused.
- Language: Go. Format with `gofmt`. Check with `go vet`. No golangci-lint unless later enabled here.
- Types: structs with JSON tags, colocated with each client package. Do not model WP/WC payloads as `map[string]any`.
- Tool JSON Schema lists only allowlisted properties.

## MVC Conventions

Controllers:

- Parse inputs.
- Call services or models.
- Return responses.
- Avoid complex business logic.
- MCP tools live in `internal/mcp`. They parse tool arguments, call a service, return sanitized content or a typed error.

Models:

- Represent persisted data and relationships.
- Contain simple domain behavior when natural.
- Avoid external API calls.
- Avoid large workflow orchestration.
- Here: request/response structs for posts, pages, media, product content. Only allowlisted fields. No DB models.

Services:

- Hold business workflows.
- Coordinate models, repositories, adapters, and jobs.
- Should be named after use cases or business actions (`UpsertPost`, `UpdateProductContent`, `UploadMedia`).

Views / Presenters / Serializers:

- Format output.
- Avoid persistence and business decisions.
- Avoid hidden data fetching when possible.
- Sanitize adapter responses before they become tool results (strip price, stock, SKU, `meta_data`, emails, detailed author).

Repositories / DAOs:

- Use when queries become complex or repeated.
- Avoid wrapping every model by default.
- Unused in this project.

## Design Pattern Conventions

Use Design Patterns only when they clarify intent.

Acceptable examples:

- `PaymentStrategy` — not used (payments are out of scope).
- `NotificationAdapter` — not used.
- `InvoiceFactory` — not used.
- `AccessPolicy` — `internal/guard` (routes, methods, fields).
- `CreateOrderCommand` — not used (orders are out of scope).
- `WordPressAdapter` / `WooCommerceAdapter` — HTTP clients.
- `AuthStrategy` — Application Password vs WC consumer key.

Avoid generic names like:

- `Manager`
- `Processor`
- `Handler`
- `Helper`
- `Util`

Unless the responsibility is very clear.

MCP SDK types may use the SDK's own `Handler` names at the `internal/mcp` boundary only.

## Naming

- Controllers should describe resources or actions (`list_posts`, `upsert_page`, `update_product_content`).
- Services should describe business workflows.
- Policies should describe permissions (`RouteAllowlist`, `ForbiddenFields`).
- Adapters should name the external system (`WordPress`, `WooCommerce`).
- Factories should name what they create.
- Strategies should name the interchangeable behavior (`ApplicationPassword`, `WooCommerceConsumer`).
- Exported identifiers: PascalCase. Unexported variables, types, and functions: camelCase.
- File names: lowercase Go (`client.go`, `post.go`, `guard.go`).

## Error Handling

- Expected business failures should be explicit.
- Unexpected errors should preserve context.
- Do not leak secrets or internal traces to users.
- Do not silently ignore failures.
- Fixed codes (do not paraphrase for the model to invent a workaround):
  - `FIELD_FORBIDDEN` — commercial, account, or non-allowlisted field in a write body. Fail the tool; do not strip and continue.
  - `ROUTE_FORBIDDEN` — path outside `my_docs/mcpContext.md` §5.
  - `METHOD_FORBIDDEN` — DELETE except coupon, PUT on product, POST to create product.
  - `DISCOUNT_CONFIRMATION_REQUIRED` — coupon percent > 20 without human AskQuestion confirmation.
  - `AUTH_MISSING` — env incomplete or absent at boot or tool time.
- Never log `WP_APP_PASSWORD`, `WC_CONSUMER_SECRET`, Authorization headers, or `.env` contents.

## Agent Rules

- Agents must keep edits aligned with MVC as mapped in `architecture_rules.md`.
- Agents must avoid introducing DDD/Clean terminology unless enabled.
- Agents must use existing framework conventions first (`gofmt`, `go vet`, official Go MCP SDK).
- Agents must not refactor into layers without explicit need.
- Agents must not add tools or writable fields outside `my_docs/mcpContext.md`.
