# feature-media-upload.md

## Feature: media-upload

### Depende de

- feature-env-auth (auth para `/wp/v2/media`)
- feature-mcp-runtime (tool `upload_media`)

### Descrição

Media (só upload para featured): POST e GET `/wp-json/wp/v2/media` e `/wp-json/wp/v2/media/{id}`. A tool `upload_media` faz upload de imagem e devolve `id`. Esse ID entra em `featured_media` (posts/páginas) ou `images` (produtos) já no media library.

Não liga a settings/logo do site.

### Problema

Posts, páginas e produtos editoriais precisam de imagens destacadas/galeria sem abrir a REST de settings, temas ou um editor visual. Sem uma tool só de media, o agente tentaria HTTP genérico ou LiveCanvas.

### Solução e trade-offs

Uma tool fechada: upload + GET por id; resposta com `id` (e o mínimo editorial necessário). `featured_media` / `images` nas outras tools só aceitam IDs já obtidos assim.

Trade-off: não há gestão de biblioteca, recorte, nem alteração do logo do site. Tamanho máximo de ficheiro: TBD em `operational_constraints.md`.

### Fluxo (given/when/then)

- Dado env válido e um ficheiro de imagem
- Quando `upload_media` corre
- Então POST `/wp/v2/media` e a tool devolve `id` (sem settings)

- Dado um `id` devolvido
- Quando `upsert_post` / `upsert_page` envia `featured_media` ou `update_product_content` envia `images`
- Então o body só referencia attachments já na library

### Casos de erro (explícitos, não deixar implícito)

- Usar media para logo/settings do site → fora de âmbito; recusar
- Path fora de `/wp/v2/media` → `ROUTE_FORBIDDEN`
- DELETE de media → `METHOD_FORBIDDEN`
- Auth em falta → `AUTH_MISSING`

### Critério de aceite (o que prova que está pronto)

- [ ] Teste: `upload_media` POST só `/wp-json/wp/v2/media` e devolve `id`
- [ ] Teste: GET `/wp/v2/media/{id}` permitido; outros paths não
- [ ] Teste: DELETE media → `METHOD_FORBIDDEN`
- [ ] Teste: a tool não chama `/wp/v2/settings` nem altera logo

### Exemplo / contexto

Tabela §5 e tool `upload_media` em `my_docs/mcpContext.md` §7: “Upload de imagem; devolve `id`. Não faz: ligar a settings/logo do site.” Posts/páginas usam `featured_media`; produtos usam `images` (IDs/src já no media library).

### Design Patterns (Gang of Four) Sugerido

- Adapter — upload multipart / GET media no client `internal/wordpress`
- Policy — rotas §5 de media; sem settings
