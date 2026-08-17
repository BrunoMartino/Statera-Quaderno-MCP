# testing_expectation.md

## Purpose

Define the testing expectations for an MVC project using pragmatic Design Patterns.

## Testing Philosophy

- Test behavior, not implementation details.
- Keep tests close to the level of risk.
- Domain invariants must be tested directly.
- Bug fixes require regression tests.
- Avoid testing framework internals.
- Runner: `go test`. External WordPress is replaced by `httptest` fakes. No network in the default suite.

## MVC Test Expectations

Controller tests:

- Verify request handling, authorization, response status, redirects, and payload shape.
- Mock or isolate external services when appropriate.
- Avoid testing complex business rules only through controllers.
- MCP tools: valid allowlisted writes succeed against the fake; missing auth returns `AUTH_MISSING`; unknown tool names do not exist.

Model tests:

- Verify validations, relationships, scopes, simple domain behavior, and persistence rules.
- Cover critical constraints.
- JSON unmarshal/marshal of allowlisted structs; extra commercial fields must not be writable types on product update bodies.

Service tests:

- Verify business workflows.
- Cover branching rules, failure modes, and side effects.
- Prefer these for non-trivial business logic.
- Upsert post/page with allowlisted fields; update product content; coupon create/list/update/delete; list orders/payments/shipments via GET; refuse create-product, refuse POST/PUT/PATCH/DELETE orders, and DELETE except coupon.

Policy / Guard tests:

- Verify access rules.
- Cover allowed and denied cases.
- PATCH with `regular_price` (or any §6.2 forbidden key) → `FIELD_FORBIDDEN`; no HTTP write.
- Path `/wp/v2/users` → `ROUTE_FORBIDDEN`; client must not send the request.
- DELETE post → `METHOD_FORBIDDEN`.
- DELETE coupon → allowed (`/wc/v3/coupons/{id}`).
- `create_coupon` / `update_coupon` with percent > 20 and no confirmation → `DISCOUNT_CONFIRMATION_REQUIRED`; no HTTP.
- Product GET/PATCH response sanitization: no price, stock, SKU, `meta_data`.
- GET `/wc/v3/orders` allowed; POST/PUT/PATCH/DELETE `/wc/v3/orders` → `METHOD_FORBIDDEN`.
- `list_orders` / `list_payments` / `list_shipments` sanitization: no email, phone, or full address.

Adapter tests:

- Verify mapping between project code and external systems.
- Avoid real external calls in regular test suites unless explicitly marked.
- Fake WP/WC: correct paths and methods from §5; Authorization header present and not logged; GET retry at most once; POST/PATCH never retried; reject redirect to another host.

Job tests:

- Verify enqueueing, idempotency, retry behavior, and service invocation.
- Unused: no jobs in this project.

## Design Pattern Test Expectations

Factory:

- Test creation rules and invalid inputs.

Strategy:

- Test each strategy independently.
- Test strategy selection separately.
- Application Password vs WC consumer key: which credentials are sent; boot fails if neither pair is complete.

Adapter:

- Test request/response mapping.
- Test external failure handling.

Command:

- Test success, validation failure, and side effects.

Policy:

- Test permission matrix clearly (routes §5, methods, field allowlists §6.1–6.2 and coupon fields).

## Minimum Requirements

Every change should include:

- Relevant automated tests.
- Regression coverage for fixed bugs.
- Tests for affected domain invariants.
- Manual verification notes only when automation is insufficient.

Required invariant cases (context checklist):

- PATCH product with `regular_price` → tool error, fake store never receives that field.
- PATCH product `description` only → adapter called, sanitized result returned.
- Guard never issues GET `/wp/v2/users`.
- GET `/wc/v3/orders` is allowed; POST/PUT/PATCH/DELETE orders is `METHOD_FORBIDDEN`.
- DELETE post/product is `METHOD_FORBIDDEN`; DELETE coupon is allowed.
- Order list responses omit email, phone, and full address.
- Percent coupon > 20 without confirmation → `DISCOUNT_CONFIRMATION_REQUIRED`.

## Agent Rules

- Agents must add focused tests for changed behavior.
- Agents must not add broad brittle tests just to increase coverage.
- Reviewer agents must flag missing invariant tests.
- Test agents must report what was run and what was not run.
- Agents must not edit or delete existing tests (project persisted-tester rule); add new tests instead.
