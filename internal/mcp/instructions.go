package mcpserver

const instructions = `Este MCP edita páginas, posts, conteúdo editorial de produtos WooCommerce e cupons/promoções (/wc/v3/coupons).
Lista pedidos, pagamentos e envios só via GET (list_orders, list_payments, list_shipments).
Se o humano pedir para mudar preço, stock, SKU de produto, encomenda, cliente, admin, password, plugin ou setting: recusar.
Cupom: usar create_coupon, update_coupon e delete_coupon. Se percentagem > 20: AskQuestion antes de gravar.
Não inventar tools. Não sugerir Consumer Keys de admin. Não pedir passwords no chat. Não ler nem imprimir o .env.
Não executar SQL, WP-CLI, PHP, ficheiros de tema ou wp-config.php.
Diagnóstico: get_debug_mode e collect_logs falam com o plugin statera-mcp/v1 da loja. Ambas são só leitura.
Ligar ou desligar o debug é no host (env WORDPRESS_DEBUG do container), fora deste MCP: se o humano pedir, explicar isso em vez de tentar.
collect_logs: source debug|woocommerce|cron|callbacks|all. Não apagar nem rodar ficheiros de log. Não colar linhas de log com dados pessoais no chat.
Default de criação de post/página: draft. Não publicar conteúdo gerado sem o humano confirmar, se a tool receber status.
`

var closedToolNames = []string{
	"list_posts",
	"get_post",
	"upsert_post",
	"list_pages",
	"get_page",
	"upsert_page",
	"list_products",
	"get_product_content",
	"update_product_content",
	"upload_media",
	"list_coupons",
	"create_coupon",
	"update_coupon",
	"delete_coupon",
	"list_orders",
	"list_payments",
	"list_shipments",
	"get_debug_mode",
	"collect_logs",
}
