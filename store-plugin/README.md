# store-plugin — Statera MCP Observability

O WordPress não expõe os seus logs por REST: `debug.log`, `wc-logs`, a fila do cron e as
entregas de webhook são ficheiros e estado em memória. Este mu-plugin é a única forma de as
tools `get_debug_mode` e `collect_logs` do MCP verem alguma coisa.

Ambas são **só leitura**. Ligar e desligar o `WP_DEBUG` é do host (`WORDPRESS_DEBUG` no
container), não deste plugin nem do MCP. Por isso **não há nada a alterar no `wp-config.php`**.

O binário Go nunca toca em ficheiros da loja: fala só HTTPS com estas duas rotas.

## Instalação

1. **Copiar o mu-plugin**

   `statera-mcp-observability.php` → `wp-content/mu-plugins/statera-mcp-observability.php`

   Os mu-plugins não precisam de activação no admin. Se `mu-plugins/` não existir, criar.

2. **Gerar um segredo**

   ```bash
   openssl rand -hex 32
   ```

3. **Variáveis de ambiente do serviço WordPress** (Coolify → o serviço → Environment Variables)

   | Variável | Valor | Para quê |
   |----------|-------|----------|
   | `WP_MCP_OBSERVABILITY_SECRET` | o segredo gerado | autentica as rotas de logs |
   | `WP_MCP_OBSERVABILITY_USER` | `mcp-content` (o login do MCP) | dá-lhe a capability sem ser admin |
   | `WORDPRESS_CONFIG_EXTRA` | ver abaixo | manda os erros para ficheiro em vez do ecrã |
   | `WORDPRESS_DEBUG` | `1` só quando estás a diagnosticar | liga o `WP_DEBUG` |

   `WORDPRESS_CONFIG_EXTRA`, numa linha (caminho absoluto, ver aviso abaixo):

   ```php
   define('WP_DEBUG_LOG', '/var/www/html/wp-content/statera-mcp-logs/debug.log'); define('WP_DEBUG_DISPLAY', false); @ini_set('display_errors', '0');
   ```

   **Isto não é opcional.** `WORDPRESS_DEBUG=1` sozinho liga o `WP_DEBUG`, mas no WordPress o
   `WP_DEBUG_LOG` tem default `false` e o `WP_DEBUG_DISPLAY` tem default **`true`**: os erros
   iriam para o ecrã dos visitantes e não para ficheiro nenhum. Estas três instruções invertem
   isso. Podem ficar sempre definidas — não fazem nada enquanto o `WP_DEBUG` estiver off.

   ⚠️ **Caminho absoluto, não `WP_CONTENT_DIR`.** O `WORDPRESS_CONFIG_EXTRA` é avaliado dentro
   do `wp-config.php`, e nessa altura nem `WP_CONTENT_DIR` nem `ABSPATH` existem ainda (são
   definidos depois, no `wp-settings.php`). Usar `WP_CONTENT_DIR` ali dá
   `Uncaught Error: Undefined constant` em PHP 8 — o site não arranca. `/var/www/html` é a raiz
   da imagem oficial do WordPress; confirmar se a tua imagem usar outra.

4. **O mesmo segredo no `.env` do MCP**

   ```
   WP_MCP_OBSERVABILITY_SECRET=<o mesmo valor>
   ```

5. **Confirmar**

   Pedir ao agente `get_debug_mode`. Espera-se `wp_debug_log: true`, `wp_debug_display: false`
   e `log_writable: true`. O campo `notice` diz o que falta quando algo não bate certo.

> O diretório `wp-content/statera-mcp-logs/` é criado pelo mu-plugin no primeiro pedido, com
> `.htaccess` (deny all) e `index.php`. Se ligares o debug antes de o plugin ter corrido uma
> vez, o PHP não consegue escrever o log até o diretório existir.

## Ciclo de diagnóstico

1. `WORDPRESS_DEBUG=1` no Coolify → redeploy/restart do serviço.
2. Reproduzir o problema.
3. `collect_logs` com `source=debug` (ou `all`).
4. `WORDPRESS_DEBUG` de volta a vazio → redeploy.

O passo 1 e o 4 são do MCP do Coolify, não deste. `get_debug_mode` serve para confirmar que o
passo 1 fez efeito antes de perderes tempo a reproduzir.

## O que as rotas fazem

| Rota | Método | O que devolve |
|------|--------|---------------|
| `/wp-json/statera-mcp/v1/debug` | GET | `WP_DEBUG`, `WP_DEBUG_LOG`, `WP_DEBUG_DISPLAY`, caminho/tamanho do log, `notice` |
| `/wp-json/statera-mcp/v1/logs` | GET | entradas de `source`, `level`, `since`, `limit`, `search` |

Não existe método de escrita em nenhuma das duas. `POST`/`PUT`/`PATCH`/`DELETE` são recusados
pelo MCP antes de sair um pedido, e não têm handler no plugin.

Autenticação: Application Password (Basic) **e** o header `X-Statera-MCP-Secret`, **e** a
capability `statera_mcp_observability` (ou `manage_options`). Segredo em falta na loja → 503;
segredo errado ou sem capability → 403.

## Garantias de segurança

- **Só leitura.** Não há rota para apagar, rodar ou escrever logs, nem para ler ficheiros
  arbitrários: `source` é um enum fechado e não existe parâmetro de caminho.
- **Logs fora da web.** `wp-content/statera-mcp-logs/` com `.htaccess` deny e `index.php`.
- **Sanitização na origem.** Emails, telefones, IPs, cartões, `Authorization`, cookies, tokens
  e chaves `ck_`/`cs_` saem redigidos — e o MCP repete a limpeza do lado de Go.
- **Tecto por resposta.** 2000 caracteres por linha, 500 entradas, e leitura só da cauda dos
  ficheiros (256 KB do debug, 128 KB por ficheiro WC, no máximo 8 ficheiros).
- **Credencial separada.** O segredo é independente da Application Password: uma credencial de
  conteúdo roubada não lê logs.

## Fontes de log

| `source` | Origem |
|----------|--------|
| `debug` | `WP_DEBUG_LOG`, `statera-mcp-logs/debug.log`, `wp-content/debug.log` |
| `woocommerce` | `wp-content/uploads/wc-logs/*.log` (8 ficheiros mais recentes) |
| `cron` | `_get_cron_array()` (hook, schedule, próxima execução, atrasados), `DISABLE_WP_CRON`, e Action Scheduler se existir |
| `callbacks` | `statera-mcp-logs/callbacks.log`, escrito pelo plugin |
| `all` | as quatro, ordenadas por data |

O log de callbacks regista **apenas falhas**: entregas de webhook WooCommerce
(`woocommerce_webhook_delivery`), chamadas HTTP de saída com erro ou status ≥ 400
(`http_api_debug`), e callbacks REST que devolveram erro em rotas de
webhook/callback/ipn/notify/payment/gateway. Rotação a 1 MB. Para desligar:

```php
define( 'STATERA_MCP_DISABLE_CALLBACK_LOG', true );
```

(ou via `WORDPRESS_CONFIG_EXTRA`, no mesmo formato do passo 3)

## Testes

```bash
php store-plugin/tests/parser_test.php
```

Cobre o parsing de `debug.log` e de `wc-logs`, o agrupamento de stack traces e a sanitização.
Não precisa de WordPress nem de rede.

## Desinstalar

Apagar `wp-content/mu-plugins/statera-mcp-observability.php` e a pasta
`wp-content/statera-mcp-logs/`. As env vars do serviço podem ficar; sem o plugin não fazem nada
(à excepção de `WORDPRESS_DEBUG`, que continua a ser o toggle do WordPress).
