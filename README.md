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
| Pedidos / pagamentos / envios (só GET) | `list_orders`, `list_payments`, `list_shipments` |
| Diagnóstico (só leitura) | `get_debug_mode`, `collect_logs` |

Promoção = cupom (`/wc/v3/coupons`), não `sale_price` no produto. Create/update de cupom com **percentagem > 20** exige confirmação humana (`human_confirmed` / AskQuestion). DELETE só de cupom; unpublish de post/página = `status: draft`.

Auth: **um** modo para WP REST e WC REST. Default: Application Password. Alternativa: par WooCommerce consumer (só se não houver `WP_APP_PASSWORD`).

## Logs e estado de debug

O WordPress não expõe os seus logs por REST: `debug.log`, `wc-logs`, a fila do cron e as entregas de webhook são ficheiros. As duas tools falam com um mu-plugin dedicado — **`store-plugin/`, instalação de uma vez por loja** ([instruções](store-plugin/README.md)), sem qualquer alteração ao `wp-config.php`. Sem ele, as tools respondem erro; o resto do MCP funciona na mesma.

| Tool | O que faz |
|------|-----------|
| `get_debug_mode` | Lê `WP_DEBUG`, `WP_DEBUG_LOG`, `WP_DEBUG_DISPLAY`, caminho e tamanho do log |
| `collect_logs` | `source` · `level` · `since` · `limit` (≤ 500) · `search` |

`source` é um enum fechado:

- `debug` — `debug.log` do WordPress (fatals, warnings, deprecations)
- `woocommerce` — ficheiros de `wp-content/uploads/wc-logs`
- `cron` — eventos WP-Cron (hook, `schedule`, próxima execução, atrasados) + Action Scheduler
- `callbacks` — entregas de webhook, HTTP de saída falhado, callbacks REST com erro
- `all` — as quatro, ordenadas por data

**Ligar e desligar o debug é do host, não deste MCP.** Em Docker/Coolify é o env var `WORDPRESS_DEBUG` do container. Este MCP só lê: não há tool de escrita, e o plugin não tem handler para ela. `get_debug_mode` serve para confirmar que o toggle fez efeito antes de perderes tempo a reproduzir o problema.

Atenção ao default do WordPress: `WORDPRESS_DEBUG=1` sozinho mostra erros aos **visitantes** e não escreve ficheiro nenhum (`WP_DEBUG_LOG` default `false`, `WP_DEBUG_DISPLAY` default `true`). O passo 3 de [store-plugin/README.md](store-plugin/README.md) corrige isso de forma permanente.

Travões do lado dos logs:

- **Só leitura.** Sem apagar, rodar ou escrever; sem parâmetro de caminho, logo sem travessia de directórios.
- **Logs fora da web.** `wp-content/statera-mcp-logs/` com `.htaccess` deny.
- **Sanitização dupla** (na loja e em Go): email, telefone, IP, cartão, `Authorization`, cookies, tokens e chaves `ck_`/`cs_` saem redigidos; 2000 caracteres por linha, 500 entradas por resposta.
- **Segredo dedicado.** `WP_MCP_OBSERVABILITY_SECRET`, separado da Application Password: credencial de conteúdo roubada não lê logs.

## O que não opera

- Preço de produto (`regular_price`, `sale_price`, stock, SKU)
- Escrever encomendas, pagamentos ou envios (POST/PUT/PATCH/DELETE)
- Clientes, users, settings, plugins, temas
- MCP nativo Woo, Store API / checkout, LiveCanvas
- DELETE de posts, páginas, produtos ou media
- Criar produto novo
- HTTP genérico, SQL, WP-CLI, PHP do tema (`wp-config`, filtros, shortcodes como código)
- Ligar ou desligar o debug (é do host: `WORDPRESS_DEBUG` do container)
- Apagar, rodar ou escrever ficheiros de log
- Ler ficheiros arbitrários da loja (`collect_logs` não tem parâmetro de caminho)
- Credenciais no chat ou nos argumentos das tools

PHP da loja (filtros no tema) é defesa **no site**, noutro repositório. Este MCP **usa a API**; não programa a loja.

## Requisitos

- Go (versão do `go.mod`)
- Loja em **HTTPS**
- User WordPress dedicado (ex. `mcp-content`), não admin

Capabilities mínimas: `read`, `edit_posts`, `edit_pages`, `publish_posts`, `publish_pages`, `upload_files`, `edit_products`, cupons (`edit_shop_coupons`, `publish_shop_coupons`, `delete_shop_coupons`), e leitura de encomendas (`read_shop_orders`). Sem `manage_options` / `manage_woocommerce` se a loja o permitir.

Para as tools de diagnóstico, mais a capability `statera_mcp_observability` — o mu-plugin atribui-a ao user indicado em `WP_MCP_OBSERVABILITY_USER`, sem precisar de admin.

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
| `WP_MCP_OBSERVABILITY_SECRET` | segredo das rotas de logs; igual ao env var do serviço WordPress |
| `DOTENV_PATH` | path absoluto do `.env` se não estiver no cwd |

A loja tem de estar em HTTPS; sem isso o menu de Application Passwords não aparece.

Se `WP_APP_PASSWORD` estiver definido, as chaves WC no mesmo ficheiro são ignoradas.

## Erros estáveis

`AUTH_MISSING` · `FIELD_FORBIDDEN` · `ROUTE_FORBIDDEN` · `METHOD_FORBIDDEN` · `DISCOUNT_CONFIRMATION_REQUIRED` · `OBSERVABILITY_SECRET_MISSING`

Campo proibido falha a tool; não há strip silencioso.

## Docs

- Contrato de domínio: `docs/harness/`
- Design: `docs/design/`
- Plugin da loja (logs): `store-plugin/README.md`

## Licença da imagem

A pintura de 1514 está em domínio público. Reprodução em `docs/assets/the-moneylender-and-his-wife.png`.
