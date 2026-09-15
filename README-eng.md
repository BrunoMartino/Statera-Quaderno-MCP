# Statera-Quaderno MCP

[Português](README.md)

<p align="center">
  <img src="docs/assets/the-moneylender-and-his-wife.png" alt="Quentin Matsys, The Moneylender and His Wife (1514)" width="720" />
</p>

<p align="center"><em>Quentin Matsys, The Moneylender and His Wife (1514) — precision and a closed scale, not an open till.</em></p>

Go MCP server for **WordPress + WooCommerce** stores. Agents (Cursor, Claude, and others) edit editorial content and coupons — one store per process, credentials only in the environment, allowlisted tools only.

This is **not** WooCommerce’s native MCP (`/woocommerce/mcp`), which exposes price, stock, and orders. This binary is a **closed** tool set.

One instance = one `.env` = one store (`WP_BASE_URL`). The same binary serves another shop by swapping env only.

## What it does

| Area | Tools |
|------|--------|
| Posts | `list_posts`, `get_post`, `upsert_post` |
| Pages | `list_pages`, `get_page`, `upsert_page` |
| Products (copy/images) | `list_products`, `get_product_content`, `update_product_content` |
| Media | `upload_media` |
| Coupons / promotions | `list_coupons`, `create_coupon`, `update_coupon`, `delete_coupon` |
| Orders / payments / shipments (GET only) | `list_orders`, `list_payments`, `list_shipments` |
| Diagnostics (read-only) | `get_debug_mode`, `collect_logs` |

Promotion = coupon (`/wc/v3/coupons`), not product `sale_price`. Create/update with **percent > 20** requires human confirmation (`human_confirmed` / AskQuestion). DELETE is coupons only; unpublish a post/page with `status: draft`.

Auth: **one** mode for both WP REST and WC REST. Default: Application Password. Fallback: WooCommerce consumer key pair (only if `WP_APP_PASSWORD` is unset).

## Logs and debug state

WordPress does not expose its logs over REST: `debug.log`, `wc-logs`, the cron queue and webhook deliveries are files. These two tools talk to a dedicated mu-plugin — **`store-plugin/`, installed once per store** ([instructions](store-plugin/README.md)), with no `wp-config.php` change at all. Without it those tools return an error; the rest of the MCP keeps working.

| Tool | What it does |
|------|--------------|
| `get_debug_mode` | Reads `WP_DEBUG`, `WP_DEBUG_LOG`, `WP_DEBUG_DISPLAY`, log path and size |
| `collect_logs` | `source` · `level` · `since` · `limit` (≤ 500) · `search` |

`source` is a closed enum:

- `debug` — WordPress `debug.log` (fatals, warnings, deprecations)
- `woocommerce` — files under `wp-content/uploads/wc-logs`
- `cron` — WP-Cron events (hook, `schedule`, next run, overdue) plus Action Scheduler
- `callbacks` — webhook deliveries, failed outbound HTTP, REST callbacks that errored
- `all` — all four, newest first

**Turning debug on and off belongs to the host, not to this MCP.** On Docker/Coolify that is the container's `WORDPRESS_DEBUG` env var. This MCP only reads: there is no write tool, and the plugin has no handler for one. `get_debug_mode` exists to confirm the toggle took effect before you spend time reproducing the bug.

Mind the WordPress defaults: `WORDPRESS_DEBUG=1` on its own shows errors to **visitors** and writes no file at all (`WP_DEBUG_LOG` defaults to `false`, `WP_DEBUG_DISPLAY` defaults to `true`). Step 3 of [store-plugin/README.md](store-plugin/README.md) fixes that permanently.

Brakes on the log side:

- **Read-only.** No delete, rotate, or write; no path parameter, hence no directory traversal.
- **Logs out of the web root.** `wp-content/statera-mcp-logs/` with a deny-all `.htaccess`.
- **Double sanitization** (store side and Go side): email, phone, IP, PAN, `Authorization`, cookies, tokens and `ck_`/`cs_` keys come back redacted; 2000 characters per line, 500 entries per response.
- **Dedicated secret.** `WP_MCP_OBSERVABILITY_SECRET`, separate from the Application Password: a stolen content credential cannot read logs.

## What it does not do

- Product price (`regular_price`, `sale_price`, stock, SKU)
- Writing orders, payments, or shipments (POST/PUT/PATCH/DELETE)
- Customers, users, settings, plugins, themes
- Native Woo MCP, Store API / checkout, LiveCanvas
- DELETE of posts, pages, products, or media
- Creating a new product
- Generic HTTP, SQL, WP-CLI, theme PHP (`wp-config`, filters, shortcode handlers)
- Turning debug on or off (that is the host's `WORDPRESS_DEBUG`)
- Deleting, rotating, or writing log files
- Reading arbitrary store files (`collect_logs` has no path parameter)
- Credentials in chat or tool arguments

Store-side PHP (theme filters) is defense **on the site**, in another repo. This MCP **calls the API**; it does not program the shop.

## Requirements

- Go (see `go.mod`)
- Store on **HTTPS**
- Dedicated WordPress user (e.g. `mcp-content`), not an administrator

Minimum capabilities: `read`, `edit_posts`, `edit_pages`, `publish_posts`, `publish_pages`, `upload_files`, `edit_products`, coupon caps (`edit_shop_coupons`, `publish_shop_coupons`, `delete_shop_coupons`), and order read (`read_shop_orders`). Avoid `manage_options` / `manage_woocommerce` when the shop allows it.

The diagnostics tools additionally need the `statera_mcp_observability` capability — the mu-plugin grants it to the user named in `WP_MCP_OBSERVABILITY_USER`, so the MCP user still does not need to be an administrator.

## Install

```bash
git clone <repo>
cd statera-quaderno-mcp
go test ./...
cp .env.example .env
```

Fill `.env` (never commit it). Boot refuses incomplete auth (`AUTH_MISSING`).

```bash
# stdio (Cursor)
go run ./cmd/statera-quaderno-mcp

# Streamable HTTP (e.g. Coolify)
go run ./cmd/statera-quaderno-mcp -http :8080
```

Cursor: one MCP server per store, each with its own `envFile`.

```json
{
  "mcpServers": {
    "store-staging": {
      "command": "go",
      "args": ["run", "./cmd/statera-quaderno-mcp"],
      "envFile": ".env"
    }
  }
}
```

Coolify: service env vars = `.env` contents; do not bake `.env` into the image.

## Application Password → `.env`

Do not use the WordPress login password.

1. Create the dedicated user (e.g. `mcp-content`).
2. **Users → that user → Application Passwords**  
   (on your own profile: **Users → Profile → Application Passwords**, at the bottom of the page.)
3. Name the key (e.g. `mcp`), generate it, copy the `xxxx` groups **immediately** — WordPress shows them once.
4. Put them in `.env`:

| Variable | Role |
|----------|------|
| `WP_BASE_URL` | HTTPS origin, no trailing slash |
| `WP_APP_USER` | user login |
| `WP_APP_PASSWORD` | application password |
| `WP_STORE_ID` | store slug (logs) |
| `WP_ENVIRONMENT` | `local` \| `staging` \| `production` |
| `WP_MCP_ALLOWED_STATUSES` | default `draft,publish,pending` |
| `WP_MCP_USER_LOGIN` | default = `WP_APP_USER` |
| `WC_CONSUMER_KEY` / `WC_CONSUMER_SECRET` | only if there is **no** App Password |
| `WP_MCP_OBSERVABILITY_SECRET` | secret for the log routes; must match the WordPress service env var |
| `DOTENV_PATH` | absolute `.env` path if not in cwd |

The site must be HTTPS or the Application Passwords UI will not appear.

If `WP_APP_PASSWORD` is set, WC keys in the same file are ignored.

## Stable errors

`AUTH_MISSING` · `FIELD_FORBIDDEN` · `ROUTE_FORBIDDEN` · `METHOD_FORBIDDEN` · `DISCOUNT_CONFIRMATION_REQUIRED` · `OBSERVABILITY_SECRET_MISSING`

A forbidden field fails the tool; there is no silent strip.

## Docs

- Domain contract: `docs/harness/`
- Design: `docs/design/`
- Store plugin (logs): `store-plugin/README.md`

## Image license

The 1514 painting is in the public domain. File: `docs/assets/the-moneylender-and-his-wife.png`.
