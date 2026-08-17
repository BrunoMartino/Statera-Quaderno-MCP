# feature-env-auth.md

## Feature: env-auth

### Depende de

- (nenhuma)

### Descrição

O MCP é um pacote genérico. Para apontar a uma loja: copiar `.env.example` → `.env` e preencher. Arranque recusa se faltar qualquer variável marcada obrigatória.

Não ler `wp-config.php`, não pedir chaves ao agente, não aceitar `base_url` / `password` nos argumentos das tools.

Credenciais **só** no `.env`. Nunca no git, nunca no código, nunca em tool input, nunca em logs de resposta. Versionar apenas `.env.example`. Uma instância MCP = um `.env`. Loja A, loja B, homolog e produção são quatro `.env` (ou quatro cópias do processo), nunca quatro forks do código.

### Problema

O código do MCP não contém URL, utilizador, passwords nem nomes de loja. Sem um carregamento de env obrigatório e fechado, o mesmo binário não pode servir outra loja sem vazar segredos ou misturar alvos.

### Solução e trade-offs

O processo MCP carrega **um** `.env` do cwd ou de `DOTENV_PATH`. Em Cursor/Coolify: `envFile` / env do serviço apontando para esse ficheiro.

Obrigatório Application Password **ou** o par WC consumer. Recusar boot se nenhum dos dois estiver completo.

Prioridade de auth: se `WP_APP_PASSWORD` existir, usar Basic `user:app_password`. Senão, Basic `WC_CONSUMER_KEY:WC_CONSUMER_SECRET` (query `consumer_key` / `consumer_secret` **proibida** em URLs e logs).

Trade-off: um processo não é multi-loja; mudar de alvo exige outro env/processo, não uma lista de lojas no código.

### Fluxo (given/when/then)

- Dado `.env` / env Coolify com `WP_BASE_URL` HTTPS e Application Password completo
- Quando o processo arranca
- Então a config fica disponível e as tools podem autenticar para esse único host

- Dado só `WC_CONSUMER_KEY` + `WC_CONSUMER_SECRET` (sem app password)
- Quando o processo arranca
- Então usa Basic das chaves WC, nunca as põe na query string

### Casos de erro (explícitos, não deixar implícito)

- `.env` ausente ou incompleto (nem app password nem par WC) → `AUTH_MISSING`, não arranca / tool não chama a loja
- Tool recebe `base_url` ou password → rejeitar; credenciais não vêm do agente
- Redirect HTTP para host ≠ `WP_BASE_URL` → recusar

### Critério de aceite (o que prova que está pronto)

- [ ] Teste: boot sem `WP_APP_PASSWORD` e sem par WC → `AUTH_MISSING`, zero HTTP
- [ ] Teste: boot com Application Password → Authorization Basic user:password, sem query `consumer_key`
- [ ] Teste: boot só com par WC → Basic key:secret, URL sem consumer query
- [ ] Teste: `WP_BASE_URL` é o único host permitido nessa instância
- [ ] Teste: `.env.example` documenta as variáveis; código não contém URL/password de loja

### Exemplo / contexto

`.env.example` (versionado; valores fake) em `my_docs/mcpContext.md` §4.3:

```
WP_STORE_ID=minha-loja
WP_ENVIRONMENT=staging
WP_BASE_URL=https://loja.example.com
WP_APP_USER=mcp-content
WP_APP_PASSWORD=xxxx xxxx xxxx xxxx xxxx xxxx
# WC_CONSUMER_KEY=
# WC_CONSUMER_SECRET=
WP_MCP_ALLOWED_STATUSES=draft,publish,pending
WP_MCP_USER_LOGIN=mcp-content
```

Cursor: um servidor MCP por loja no `mcp.json`, cada um com `envFile` próprio. Coolify: env vars do recurso = conteúdo do `.env`; não montar `.env` na imagem.

### Design Patterns (Gang of Four) Sugerido

- Strategy — Application Password vs par WC consumer, mesma interface de auth HTTP
- Factory — construir a config a partir do env ou falhar com `AUTH_MISSING`
