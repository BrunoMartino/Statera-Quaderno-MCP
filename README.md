# Statera-Quaderno MCP

[English](README-eng.md)

<p align="center">
  <img src="docs/assets/the-moneylender-and-his-wife.png" alt="Quentin Matsys, The Moneylender and His Wife (1514)" width="720" />
</p>

<p align="center"><em>Quentin Matsys, O Cambista e a sua Mulher (1514) — precisão, pesos e medidas, não um balcão aberto.</em></p>

Servidor MCP em Go para lojas **WordPress + WooCommerce**. Agentes (Cursor, Claude, etc.) editam conteúdo editorial e cupons — com a mesma disciplina da balança na mesa: só o que está na allowlist, um alvo por processo, credenciais só no ambiente.

Não é o MCP nativo do WooCommerce (`/woocommerce/mcp`). Esse expõe preço, stock e encomendas. Este binário é **próprio**, com tools fechadas.

Uma instância = um `.env` = uma loja (`WP_BASE_URL`). O mesmo código serve outra loja só trocando o env.

## O que opera

| Área | Tools |
|------|--------|
| Posts | `list_posts`, `get_post`, `upsert_post` |
| Páginas | `list_pages`, `get_page`, `upsert_page` |
| Produtos (texto/imagem) | `list_products`, `get_product_content`, `update_product_content` |
| Media | `upload_media` |
| Cupons / promoções | `list_coupons`, `create_coupon`, `update_coupon`, `delete_coupon` |

Promoção = cupom (`/wc/v3/coupons`), não `sale_price` no produto. Create/update de cupom com **percentagem > 20** exige confirmação humana (`human_confirmed` / AskQuestion). DELETE só de cupom; unpublish de post/página = `status: draft`.

Auth: **um** modo para WP REST e WC REST. Default: Application Password. Alternativa: par WooCommerce consumer (só se não houver `WP_APP_PASSWORD`).

## O que não opera

- Preço de produto (`regular_price`, `sale_price`, stock, SKU)
- Encomendas, clientes, pagamentos, envios
- Users, settings, plugins, temas
- MCP nativo Woo, Store API / checkout, LiveCanvas
- DELETE de posts, páginas, produtos ou media
- Criar produto novo
- HTTP genérico, SQL, WP-CLI, PHP do tema (`wp-config`, filtros, shortcodes como código)
- Credenciais no chat ou nos argumentos das tools

PHP da loja (filtros no tema) é defesa **no site**, noutro repositório. Este MCP **usa a API**; não programa a loja.

## Requisitos

- Go (versão do `go.mod`)
- Loja em **HTTPS**
- User WordPress dedicado (ex. `mcp-content`), não admin

Capabilities mínimas: `read`, `edit_posts`, `edit_pages`, `publish_posts`, `publish_pages`, `upload_files`, `edit_products`, e cupons (`edit_shop_coupons`, `publish_shop_coupons`, `delete_shop_coupons`). Sem `manage_options` / `manage_woocommerce` se a loja o permitir.

## Instalar

```bash
git clone <repo>
cd statera-quaderno-mcp
go test ./...
cp .env.example .env
```

Edita o `.env` (nunca o commits). Arranque recusa se faltar auth (`AUTH_MISSING`).

```bash
# stdio (Cursor)
go run ./cmd/statera-quaderno-mcp

# Streamable HTTP (ex. Coolify)
go run ./cmd/statera-quaderno-mcp -http :8080
```

Cursor: um servidor MCP por loja, cada um com `envFile` próprio.

```json
{
  "mcpServers": {
    "loja-staging": {
      "command": "go",
      "args": ["run", "./cmd/statera-quaderno-mcp"],
      "envFile": ".env"
    }
  }
}
```

Coolify: variáveis do serviço = conteúdo do `.env`; não montar `.env` na imagem.

## Application Password → `.env`

Não uses a password de login do WordPress.

1. Cria o user dedicado (ex. `mcp-content`).
2. **Utilizadores → esse user → Application Passwords**  
   (no perfil: **Users → Profile → Application Passwords**, no fundo da página.)
3. Nome da chave (ex. `mcp`), gerar, copiar os grupos `xxxx` **já** — o WP só mostra uma vez.
4. No `.env`:

| Variável | Função |
|----------|--------|
| `WP_BASE_URL` | origem HTTPS, sem trailing slash |
| `WP_APP_USER` | login do user |
| `WP_APP_PASSWORD` | application password |
| `WP_STORE_ID` | slug da loja (logs) |
| `WP_ENVIRONMENT` | `local` \| `staging` \| `production` |
| `WP_MCP_ALLOWED_STATUSES` | default `draft,publish,pending` |
| `WP_MCP_USER_LOGIN` | default = `WP_APP_USER` |
| `WC_CONSUMER_KEY` / `WC_CONSUMER_SECRET` | só se **não** houver App Password |
| `DOTENV_PATH` | path absoluto do `.env` se não estiver no cwd |

A loja tem de estar em HTTPS; sem isso o menu de Application Passwords não aparece.

Se `WP_APP_PASSWORD` estiver definido, as chaves WC no mesmo ficheiro são ignoradas.

## Erros estáveis

`AUTH_MISSING` · `FIELD_FORBIDDEN` · `ROUTE_FORBIDDEN` · `METHOD_FORBIDDEN` · `DISCOUNT_CONFIRMATION_REQUIRED`

Campo proibido falha a tool; não há strip silencioso.

## Docs

- Contrato de domínio: `docs/harness/`
- Design: `docs/design/`

## Licença da imagem

A pintura de 1514 está em domínio público. Reprodução em `docs/assets/the-moneylender-and-his-wife.png`.
