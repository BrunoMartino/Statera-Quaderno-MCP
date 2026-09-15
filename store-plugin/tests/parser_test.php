<?php
/**
 * Smoke test das funções puras do mu-plugin (parsing e sanitização).
 * Correr: php store-plugin/tests/parser_test.php
 * Não precisa de WordPress: define os stubs mínimos usados no carregamento.
 */

define( 'ABSPATH', '/var/www/html/' );
define( 'WP_CONTENT_DIR', '/var/www/html/wp-content' );

$GLOBALS['hooks'] = array();

function add_action( $hook, $cb, $priority = 10, $args = 1 ) {
	$GLOBALS['hooks'][] = array( $hook, is_array( $cb ) ? $cb[1] : (string) $cb );
}
function add_filter( $hook, $cb, $priority = 10, $args = 1 ) {
	add_action( $hook, $cb, $priority, $args );
}

require_once __DIR__ . '/../statera-mcp-observability.php';

$failures = 0;

function hooked( $hook, $method ) {
	foreach ( $GLOBALS['hooks'] as $h ) {
		if ( $h[0] === $hook && $h[1] === $method ) {
			return true;
		}
	}
	return false;
}

function invoke( $method, array $args = array() ) {
	$ref = new ReflectionMethod( 'Statera_MCP_Observability', $method );
	$ref->setAccessible( true );
	return $ref->invokeArgs( null, $args );
}

function check( $label, $condition, $got = null ) {
	global $failures;
	if ( $condition ) {
		echo "ok   $label\n";
		return;
	}
	$failures++;
	echo "FAIL $label";
	if ( null !== $got ) {
		echo ' -> ' . var_export( $got, true );
	}
	echo "\n";
}

/* --- boot() -------------------------------------------------------------- */
/* Regressao: o PHP faz early binding da classe, por isso uma guarda com
   class_exists() no topo do ficheiro devolveria antes de chamar boot() e o
   plugin nao registaria nada — sem erro nenhum. */

check( 'boot: classe declarada', class_exists( 'Statera_MCP_Observability' ) );
check( 'boot: registou hooks', count( $GLOBALS['hooks'] ) > 0, count( $GLOBALS['hooks'] ) );
check( 'boot: rest_api_init -> register_routes', hooked( 'rest_api_init', 'register_routes' ) );
check( 'boot: init -> ensure_log_dir', hooked( 'init', 'ensure_log_dir' ) );
check( 'boot: init -> grant_capability', hooked( 'init', 'grant_capability' ) );
check( 'boot: log de callbacks ligado', hooked( 'http_api_debug', 'log_http_failure' ) );

/* --- debug.log do WordPress ------------------------------------------------ */

$debug_raw = <<<LOG
[15-Sep-2026 10:12:00 UTC] PHP Fatal error:  Uncaught Error: Call to a member function get_id() on null in /var/www/html/wp-content/plugins/pay/pay.php on line 120
Stack trace:
#0 /var/www/html/wp-includes/class-wp-hook.php(310): Pay->handle()
[15-Sep-2026 10:13:00 UTC] PHP Warning:  Undefined array key "total" in /var/www/html/wp-content/themes/x/functions.php on line 44
[15-Sep-2026 10:14:00 UTC] Pedido de cliente joao.silva@example.com falhou, tel +351 912 345 678, ip 10.0.0.9
LOG;

$entries = invoke( 'parse_php_log', array( $debug_raw, 'debug', 'php' ) );
check( 'debug: tres entradas (stack trace junta-se a primeira)', 3 === count( $entries ), count( $entries ) );
check( 'debug: fatal -> critical', 'critical' === $entries[0]['level'], $entries[0]['level'] );
check( 'debug: stack trace anexado', false !== strpos( $entries[0]['message'], 'class-wp-hook.php' ) );
check( 'debug: contexto ficheiro:linha', 'wp-content/plugins/pay/pay.php:120' === $entries[0]['context'], $entries[0]['context'] );
check( 'debug: warning -> warning', 'warning' === $entries[1]['level'], $entries[1]['level'] );
check( 'debug: timestamp ISO', '2026-09-15T10:12:00+00:00' === $entries[0]['timestamp'], $entries[0]['timestamp'] );
check( 'debug: email removido', false === strpos( $entries[2]['message'], 'joao.silva@example.com' ), $entries[2]['message'] );
check( 'debug: telefone removido', false === strpos( $entries[2]['message'], '912 345 678' ), $entries[2]['message'] );
check( 'debug: ip removido', false === strpos( $entries[2]['message'], '10.0.0.9' ), $entries[2]['message'] );

/* --- logs do WooCommerce --------------------------------------------------- */

$wc_raw = <<<LOG
2026-09-15T10:12:00+00:00 CRITICAL Uncaught exception no gateway
2026-09-15T10:13:00+00:00 INFO Pedido 727 pago com token=abcdef123456
LOG;

$wc = invoke( 'parse_wc_log', array( $wc_raw, 'fatal-errors' ) );
check( 'wc: duas entradas', 2 === count( $wc ), count( $wc ) );
check( 'wc: nivel em minusculas', 'critical' === $wc[0]['level'], $wc[0]['level'] );
check( 'wc: channel preservado', 'fatal-errors' === $wc[0]['channel'], $wc[0]['channel'] );
check( 'wc: token removido', false === strpos( $wc[1]['message'], 'abcdef123456' ), $wc[1]['message'] );

/* --- sanitização ----------------------------------------------------------- */

$dirty = 'Authorization: Basic Y2s6Y3MxMjM0 cartao 4111 1111 1111 1111 chave ck_' . str_repeat( 'a', 30 );
$clean = invoke( 'scrub', array( $dirty ) );
check( 'scrub: authorization', false === strpos( $clean, 'Y2s6Y3MxMjM0' ), $clean );
check( 'scrub: cartao', false === strpos( $clean, '4111 1111 1111 1111' ), $clean );
check( 'scrub: consumer key', false === strpos( $clean, 'ck_aaaa' ), $clean );

/* --- níveis ---------------------------------------------------------------- */

check( 'level: deprecated -> notice', 'notice' === invoke( 'level_from_message', array( 'PHP Deprecated:  x' ) ) );
check( 'level: generico -> info', 'info' === invoke( 'level_from_message', array( 'Cron disparado' ) ) );

echo $failures ? "\n$failures teste(s) falharam\n" : "\nTodos os testes passaram\n";
exit( $failures ? 1 : 0 );
