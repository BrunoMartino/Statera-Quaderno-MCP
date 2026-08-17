# deployment_rules.md

## Purpose

Define deployment, migration, rollout, rollback, and production safety rules.

## Release Principles

- Deployments must be reproducible.
- Configuration must be environment-based.
- Secrets must never be committed. Version `.env.example` only. Never write `.env` in git or in the image.
- Risky changes should use feature flags. This project has no feature flags; mitigate by not deploying the binary / disabling the MCP server entry.
- Rollback or mitigation must be known before production release.
- Same binary for every store. Loja A vs B, staging vs production = different env (`WP_BASE_URL`, user, password), never a code fork.
- Cursor: one MCP server per store in `mcp.json`, each with its own `envFile`.
- Coolify: service env vars = contents of the store `.env`; Streamable HTTP; do not mount `.env` into the image.

## Pre-Deployment Checklist

- Tests pass (`go test`).
- Lint/type checks pass (`gofmt`, `go vet`).
- Database migrations reviewed. N/A — this process has no database.
- Feature flags configured when needed. N/A.
- Observability added for risky workflows (errors without secrets; `WP_STORE_ID` / `WP_ENVIRONMENT` in logs).
- Rollback plan documented for high-risk changes.
- Boot fails closed if Application Password **or** WC consumer pair is missing (`AUTH_MISSING`).
- `WP_BASE_URL` is HTTPS, no trailing slash, single host.

## MVC Deployment Considerations

Controllers:

- Verify routes, auth, and response compatibility.
- Closed tool list unchanged across a release unless a feature doc was updated first.

Models:

- Verify migrations, validations, indexes, and data compatibility.
- N/A for persistence. Verify JSON field allowlists still match `my_docs/mcpContext.md` §6.

Services:

- Verify business workflow changes and side effects.
- Writes go only to the configured store. Mass edits: use staging env first.

Jobs:

- Verify queue compatibility, retries, and idempotency.
- N/A.

Adapters:

- Verify external credentials, timeouts, and failure behavior.
- Timeout 10s. No retry on POST/PATCH. GET at most one retry. No redirect to another host.

## Database Rules

- Prefer forward-compatible migrations.
- Avoid destructive schema changes in the same release that depends on them.
- Use expand-and-contract for risky changes.
- Backfills must be idempotent or resumable.
- Long migrations require monitoring.
- This MCP has no database. Do not add one. WordPress/MySQL stays on the store, outside this repo.

## Rollback Rules

- Code rollback must not break newer data.
- Migrations must document rollback safety. N/A.
- External side effects may require compensating actions (a published page stays published on the store; rollback of the MCP does not undo WP writes — unpublish via `status: draft` if needed).
- Feature flags should support fast mitigation. Mitigation: stop the Coolify service or remove the Cursor MCP entry.
- Rollback = previous image/binary. Env vars stay with the store instance.

## Agent Rules

- Agents must not deploy unless explicitly asked.
- Agents must surface deployment risk before acting.
- Agents must not change production config or secrets without approval.
- Orchestrators must separate build, verification, deployment, and post-deploy checks.
- Agents must not create, edit, or commit `.env`.
- PHP filters on the store (context §8) are out of this pipeline.
