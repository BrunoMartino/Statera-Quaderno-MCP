# TDD: media-upload

## 1. Header & Metadata

| Field | Value |
|-------|-------|
| Title | media-upload — upload de imagem para a media library |
| Status | Draft |
| Date | 2026-08-17 |
| Last updated | 2026-08-17 |
| Tech lead | TBD |
| Team | TBD |
| Epic / ticket | TBD |
| Size | Small |
| Type | Integration |
| Depends on | env-auth, mcp-runtime |
| Source harness | `docs/harness/features/feature-media-upload.md` |

## 2. Technical Solution

**Design pattern (escolhido):** Policy — só `/wp/v2/media` GET/POST; sem settings/logo; sem DELETE.

```mermaid
flowchart LR
  upload[upload_media]
  policy[Policy media]
  media["POST /wp/v2/media"]
  id[attachment id]
  upload --> policy
  policy --> media
  media --> id
```

### HTTP contract

| Tool | Method | Path | Result |
|------|--------|------|--------|
| `upload_media` | POST | `/wp-json/wp/v2/media` | `id` (+ mínimo editorial) |
| (leitura) | GET | `/wp-json/wp/v2/media/{id}` | metadados necessários ao id |

Consumidores: `featured_media` em posts/páginas; `images` em produto — só IDs já na library.

Tamanho máximo de ficheiro: TBD (operational constraints). Timeout 10s; POST sem retry.

## 3. Context Pillars

Fonte: harness `feature-media-upload.md` (verbatim).

### 1 — Como descreve a feature?

Media (só upload para featured): POST e GET `/wp-json/wp/v2/media` e `/wp-json/wp/v2/media/{id}`. A tool `upload_media` faz upload de imagem e devolve `id`. Esse ID entra em `featured_media` (posts/páginas) ou `images` (produtos) já no media library.

Não liga a settings/logo do site.

### 2 — Qual problema objetivamente ela resolve?

Posts, páginas e produtos editoriais precisam de imagens destacadas/galeria sem abrir a REST de settings, temas ou um editor visual. Sem uma tool só de media, o agente tentaria HTTP genérico ou LiveCanvas.

### 3 — Qual a solução esperada? Quais trade-offs ela envolve?

Uma tool fechada: upload + GET por id; resposta com `id` (e o mínimo editorial necessário). `featured_media` / `images` nas outras tools só aceitam IDs já obtidos assim.

Trade-off: não há gestão de biblioteca, recorte, nem alteração do logo do site. Tamanho máximo de ficheiro: TBD em `operational_constraints.md`.

### 4 — Qual exemplo ou contexto temos do problema e da solução?

Tabela §5 e tool `upload_media` em `my_docs/mcpContext.md` §7: “Upload de imagem; devolve `id`. Não faz: ligar a settings/logo do site.” Posts/páginas usam `featured_media`; produtos usam `images` (IDs/src já no media library).

## 4. Context

Conteúdo editorial precisa de imagens. Settings e logo do site são operações de admin, não deste MCP.

## 5. Problem Statement & Motivation

- Sem tool fechada, o agente usa HTTP genérico ou LiveCanvas.
- POST em `/settings` altera identidade da loja.
- DELETE de media perde assets.

Quantificação: TBD.

## 6. Scope

In scope (V1):

- POST media; GET por id; devolver `id`
- IDs usáveis em featured/galeria

Out of scope:

- `/wp/v2/settings`; logo do site
- DELETE media; recorte; gestão de biblioteca
- LiveCanvas

Future (V2+):

- TBD: limite de bytes
- TBD: tipos MIME allowlisted
- TBD: alt text editorial

## 7. Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Upload para settings | H | L | Policy: só `/media` |
| Payload enorme | M | M | TBD max size |
| DELETE media | H | L | `METHOD_FORBIDDEN` |

## 8. Implementation Plan

| Phase | Task | TDD cycle | Owner | Estimate |
|-------|------|-----------|-------|----------|
| 1 | POST devolve id | Red: só path media → Green | TBD | 0.5d |
| 2 | GET id; deny settings/DELETE | Red: `ROUTE`/`METHOD_FORBIDDEN` → Green | TBD | 0.5d |

## 9. Security Considerations

- Authn: env-auth.
- Authz: Policy media only.
- Sem PII extra; não logar bytes do ficheiro.
- Multipart só para media.

## 10. Testing Strategy

| Type | Scope | Approach |
|------|-------|----------|
| Unit | Policy paths | — |
| Integration | POST fake; sem `/settings` | httptest |

Cenários: upload → id; GET `/{id}`; DELETE forbidden; settings never called.

## 16. Dependencies

- env-auth, mcp-runtime.
- wordpress-content e product-content consomem o `id`.
- WP REST media na loja.
