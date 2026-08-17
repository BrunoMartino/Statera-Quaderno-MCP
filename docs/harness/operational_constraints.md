# operational_constraints.md

## Purpose

Define the runtime, security, reliability, and operational limits of the system.

## Runtime Constraints

- Maximum request latency: 10s per outbound HTTPS call to the store. MCP tool wall time = that call plus local guard/schema.
- Maximum job duration: N/A (no jobs).
- Maximum payload size: TBD: max media upload bytes (ask before implementing `upload_media` limits).
- Rate limits: none imposed by this process. TBD: store/host REST limits if a shop documents them.
- Memory limits: TBD: Coolify service memory if set at deploy time.
- CPU limits: TBD: Coolify service CPU if set at deploy time.
- Supported regions: wherever the binary runs (Cursor local or Coolify). The store region is `WP_BASE_URL`.
- Supported time zones: store timestamps as returned by WP (`modified`, GMT fields not rewritten). Process timezone does not change content.

## MVC Operational Constraints

Controllers:

- Must respond within expected latency.
- Must not perform long-running work synchronously.
- Must validate input before invoking workflows.
- Closed tool set only. Default create `status` is `draft`. Do not `publish` generated content without human confirmation when `status` is passed.

Services:

- Must handle known business failures explicitly (`FIELD_FORBIDDEN`, `ROUTE_FORBIDDEN`, `METHOD_FORBIDDEN`, `AUTH_MISSING`, `DISCOUNT_CONFIRMATION_REQUIRED`).
- Must preserve domain invariants.
- Must be safe under retry when applicable (writes are not retried here).

Models:

- Must respect database constraints.
- Must avoid hidden expensive queries where possible.
- No local DB. Structs only.

Jobs:

- Must be idempotent when retried.
- Must define retry behavior.
- Must not assume execution order unless guaranteed.
- N/A.

Adapters:

- Must define timeout behavior: 10s.
- Must define retry behavior: GET at most one retry; POST/PATCH zero retries.
- Must define fallback or failure mode: return the error to the tool; no alternate host; no query-string consumer keys.

## Data Constraints

- Data retention: this process stores nothing. Retention is the WordPress site.
- Data residency: the store host (`WP_BASE_URL`).
- PII handling: do not return emails or detailed author. Order/payment/shipment lists must not return email, phone, or full address. Do not log credentials or Authorization headers.
- Encryption requirements: HTTPS to `WP_BASE_URL` only. Application Password / WC secret in env, not in git.
- Backup requirements: N/A for this binary. Store backups are the shop’s.
- Restore expectations: N/A. Redeploy binary + env.

## External Service Constraints

Service:

- Name: WordPress REST (`/wp-json/wp/v2/posts|pages|media`)
- Purpose: list/get/upsert posts and pages; upload and get media
- Rate limits: TBD: per-store host
- Timeout: 10s
- Retry policy: GET ≤ 1 retry; POST/PATCH none
- Failure mode: tool error; no retry loop
- Fallback: none
- Owner: the shop that owns `WP_BASE_URL`

Service:

- Name: WooCommerce REST (`/wp-json/wc/v3/products`)
- Purpose: list/get/PATCH product editorial content
- Rate limits: TBD: per-store host
- Timeout: 10s
- Retry policy: GET ≤ 1 retry; PATCH none
- Failure mode: tool error
- Fallback: none
- Owner: the shop that owns `WP_BASE_URL`

Service:

- Name: WooCommerce REST coupons (`/wp-json/wc/v3/coupons`)
- Purpose: list/create/update/delete discount coupons (promoções)
- Rate limits: TBD: per-store host
- Timeout: 10s
- Retry policy: GET ≤ 1 retry; POST/PATCH/DELETE none
- Failure mode: tool error; percent > 20 without confirmation → `DISCOUNT_CONFIRMATION_REQUIRED`
- Fallback: none
- Owner: the shop that owns `WP_BASE_URL`

Service:

- Name: WooCommerce REST orders (`/wp-json/wc/v3/orders`, optional `/wp-json/wc/v3/refunds`)
- Purpose: GET-only list projections for orders, payments, and shipments (`feature-woocommerce-orders-payments-shipments-read`)
- Rate limits: TBD: per-store host
- Timeout: 10s
- Retry policy: GET ≤ 1 retry; POST/PUT/PATCH/DELETE never issued
- Failure mode: tool error; write methods → `METHOD_FORBIDDEN`
- Fallback: none
- Owner: the shop that owns `WP_BASE_URL`

Auth priority: if `WP_APP_PASSWORD` is set, Basic `WP_APP_USER:WP_APP_PASSWORD`. Else Basic `WC_CONSUMER_KEY:WC_CONSUMER_SECRET`. Never put consumer credentials in the URL.

## Security Constraints

- Authentication model: server-to-server Basic from env. No cookies. No CORS. Do not expose this MCP in an editor browser.
- Authorization model: closed tools + `internal/guard` allowlists. WP user should be a dedicated editor without `manage_woocommerce` / `manage_options` (configured on the store, not in this repo).
- Secret management: Cursor `envFile` or Coolify service env. `.env.example` versioned with placeholders. Never commit `.env`.
- Audit logging: log tool name, resource id, `WP_STORE_ID`, `WP_ENVIRONMENT`. Never log secrets or raw write bodies that might contain credentials.
- Sensitive data rules: INV-002 and INV-008. `status` allowlist default `draft,publish,pending`. Never `private` with password.

## Observability

- Logs must include request or correlation IDs.
- Metrics must cover critical workflows. TBD: Coolify/process metrics if added later; not required for v1 Cursor stdio.
- Errors must include useful context without leaking secrets (fixed error codes).
- Alerts must map to actionable ownership. TBD: Coolify health on Streamable HTTP if a probe is defined at deploy.

## Multi-Agent Constraints

- Orchestrators must classify risk before delegating.
- Agents must not assume production access. Prefer staging `WP_ENVIRONMENT` for bulk edits.
- Agents must stop before irreversible actions unless approved (DELETE só cupom; publish only with human confirmation).
- MCP outputs must be treated as observations, not guaranteed truth.
- If a human asks to **change** product price, stock, order, customer, admin, plugin, or setting: refuse. Listing orders/payments/shipments uses GET tools when allowlisted. Cupons/promoções usam as tools de cupom; percentagem > 20 exige AskQuestion.
