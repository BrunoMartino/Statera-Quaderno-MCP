# feature-wordpress-content.md

## Feature: wordpress-content

### Depende de

- feature-env-auth (Application Password / env para `/wp/v2`)
- feature-mcp-runtime (tools `list_posts`, `get_post`, `upsert_post`, `list_pages`, `get_page`, `upsert_page`)

### Descrição

Manipular a WordPress API, com auth correcta já fornecida no env, para alterar páginas (`page`) e posts (`post`): criar ou actualizar conteúdo editorial.

Permitido no body de POST/PATCH: `title`, `content`, `excerpt`, `slug`, `status` (`draft` | `publish` | `pending`), `featured_media`, `categories`/`tags` (posts).

Páginas LiveCanvas: o HTML vive em `content.rendered` / `content.raw`. O MCP edita `content`; não usa a API LiveCanvas.

### Problema

Editores, copy e agência precisam de alterar páginas e posts sem aceder a users, settings, plugins, temas, comentários ou DELETE. A REST da loja é deny-by-default; o MCP autentica-se servidor-a-servidor e não pode ser um cliente WP aberto.

### Solução e trade-offs

Cliente HTTP só para `/wp-json/wp/v2/posts` e `/wp-json/wp/v2/pages` (GET, POST, PATCH) e `/{id}`. Structs allowlisted. Leitura devolve `id`, `type`, `link`, `title`, `content`, `excerpt`, `slug`, `status`, `featured_media`, `modified`. Não devolver `author` detalhado nem emails.

Unpublish = `status: draft` via PATCH. DELETE: nunca. Criar post/página: permitido. Default de criação: `draft`. Não publicar conteúdo gerado sem o humano confirmar, se a tool receber `status`.

Trade-off: sem meta livre, ACF, template PHP, `password`, `private`. Homolog primeiro em alterações em massa.

### Fluxo (given/when/then)

- Dado env válido e um post existente
- Quando `get_post` / `upsert_post` com campos §6.1
- Então lê ou grava só esses campos e devolve o subconjunto de leitura

- Dado um pedido de nova página
- Quando `upsert_page` sem `status`
- Então cria como `draft`

### Casos de erro (explícitos, não deixar implícito)

- Body com `author`, `password`, `meta` genérico, `acf` sem allowlist, `template`, ou chave não listada → `FIELD_FORBIDDEN`, não aplica o resto
- DELETE → `METHOD_FORBIDDEN`
- Path `/wp/v2/users` (ou outro fora de §5) → `ROUTE_FORBIDDEN`
- `status` fora de `draft|publish|pending` (incl. `private` com password) → recusar

### Critério de aceite (o que prova que está pronto)

- [ ] Teste: `upsert_post` / `upsert_page` com title+content → HTTP POST/PATCH só com allowlist
- [ ] Teste: body com `password` ou `author` → `FIELD_FORBIDDEN`, fake WP não recebe o body
- [ ] Teste: `get_post` não inclui email nem author detalhado
- [ ] Teste: DELETE post → `METHOD_FORBIDDEN`
- [ ] Teste: criação sem status → `draft`
- [ ] Teste: GET `/wp/v2/users` nunca é emitido

### Exemplo / contexto

Tabela §5 e §6.1 / tools §7 em `my_docs/mcpContext.md`. Pedido original: manipular a WordPress API dado que as auth correctas tenham sido fornecidas.

Paths:

- GET, POST, PATCH `/wp-json/wp/v2/posts`, `/wp-json/wp/v2/posts/{id}`
- GET, POST, PATCH `/wp-json/wp/v2/pages`, `/wp-json/wp/v2/pages/{id}`

### Design Patterns (Gang of Four) Sugerido

- Adapter — `internal/wordpress` mapeia structs allowlisted para WP REST
- Policy — `internal/guard` nos campos §6.1 e rotas §5
