<?php
/**
 * Plugin Name: Statera MCP Observability
 * Description: Rotas REST statera-mcp/v1 usadas pelo Statera Quaderno MCP: leitura do estado de debug e recolha de logs (debug, WooCommerce, cron/schedules, callbacks). Só leitura; não liga debug, não altera conteúdo, encomendas nem settings.
 * Version: 1.0.0
 * Author: Statera
 * Requires PHP: 7.4
 *
 * Instalação: copiar este ficheiro para wp-content/mu-plugins/. Não requer
 * qualquer alteração ao wp-config.php. Ligar/desligar o WP_DEBUG é do host
 * (env WORDPRESS_DEBUG do container). Ver store-plugin/README.md.
 */

if ( ! defined( 'ABSPATH' ) ) {
	exit;
}

// Guarda de carregamento duplo. NÃO usar class_exists() aqui: o PHP faz early binding
// da classe declarada abaixo no momento da compilação, por isso class_exists() seria
// sempre true e este ficheiro devolveria antes de chamar boot().
if ( defined( 'STATERA_MCP_OBSERVABILITY_LOADED' ) ) {
	return;
}
define( 'STATERA_MCP_OBSERVABILITY_LOADED', true );

final class Statera_MCP_Observability {

	const VERSION  = '1.0.0';
	const REST_NS  = 'statera-mcp/v1';
	const SECRET_HEADER = 'x-statera-mcp-secret';

	const CAPABILITY = 'statera_mcp_observability';

	const LIMIT_MAX           = 500;
	const LIMIT_DEFAULT       = 100;

	const DEBUG_TAIL_BYTES    = 262144;
	const WC_TAIL_BYTES       = 131072;
	const WC_MAX_FILES        = 8;
	const CALLBACK_LOG_MAX    = 1048576;
	const MESSAGE_MAX_CHARS   = 2000;

	public static function boot() {
		add_action( 'rest_api_init', array( __CLASS__, 'register_routes' ) );
		add_action( 'init', array( __CLASS__, 'grant_capability' ) );
		add_action( 'init', array( __CLASS__, 'ensure_log_dir' ) );

		if ( ! defined( 'STATERA_MCP_DISABLE_CALLBACK_LOG' ) || ! STATERA_MCP_DISABLE_CALLBACK_LOG ) {
			add_action( 'woocommerce_webhook_delivery', array( __CLASS__, 'log_webhook_delivery' ), 10, 5 );
			add_action( 'http_api_debug', array( __CLASS__, 'log_http_failure' ), 10, 5 );
			add_filter( 'rest_request_after_callbacks', array( __CLASS__, 'log_rest_error' ), 10, 3 );
		}
	}

	/* ---------------------------------------------------------------- paths */

	private static function log_dir() {
		return WP_CONTENT_DIR . '/statera-mcp-logs';
	}

	private static function debug_log_file() {
		return self::log_dir() . '/debug.log';
	}

	private static function callback_log_file() {
		return self::log_dir() . '/callbacks.log';
	}

	/**
	 * Cria o diretório de logs fechado ao público (Apache + fallback de índice).
	 */
	public static function ensure_log_dir() {
		$dir = self::log_dir();
		if ( ! is_dir( $dir ) ) {
			wp_mkdir_p( $dir );
		}
		if ( ! is_dir( $dir ) ) {
			return false;
		}
		$htaccess = $dir . '/.htaccess';
		if ( ! file_exists( $htaccess ) ) {
			@file_put_contents( $htaccess, "Order allow,deny\nDeny from all\n<IfModule mod_authz_core.c>\nRequire all denied\n</IfModule>\n" );
		}
		$index = $dir . '/index.php';
		if ( ! file_exists( $index ) ) {
			@file_put_contents( $index, "<?php // Silence is golden.\n" );
		}
		return true;
	}

	/* ----------------------------------------------------------------- auth */

	private static function secret() {
		if ( defined( 'STATERA_MCP_OBSERVABILITY_SECRET' ) && STATERA_MCP_OBSERVABILITY_SECRET ) {
			return (string) STATERA_MCP_OBSERVABILITY_SECRET;
		}
		$env = getenv( 'WP_MCP_OBSERVABILITY_SECRET' );
		return is_string( $env ) ? trim( $env ) : '';
	}

	/**
	 * Segredo dedicado + capability. Uma credencial de conteúdo roubada não chega para ler logs.
	 */
	public static function authorize( $request ) {
		$secret = self::secret();
		if ( '' === $secret ) {
			return new WP_Error(
				'observability_secret_missing',
				'STATERA_MCP_OBSERVABILITY_SECRET não está definido na loja.',
				array( 'status' => 503 )
			);
		}
		$sent = (string) $request->get_header( self::SECRET_HEADER );
		if ( '' === $sent || ! hash_equals( $secret, $sent ) ) {
			return new WP_Error( 'observability_forbidden', 'Segredo de observabilidade inválido.', array( 'status' => 403 ) );
		}
		if ( ! current_user_can( self::CAPABILITY ) && ! current_user_can( 'manage_options' ) ) {
			return new WP_Error(
				'observability_forbidden',
				'Utilizador sem a capability ' . self::CAPABILITY . '.',
				array( 'status' => 403 )
			);
		}
		return true;
	}

	/**
	 * Dá a capability de observabilidade ao user do MCP definido em
	 * STATERA_MCP_OBSERVABILITY_USER, para que ele não precise de ser admin.
	 * Corre uma vez: add_cap grava na base e o guard evita reescrever.
	 */
	public static function grant_capability() {
		$login = '';
		if ( defined( 'STATERA_MCP_OBSERVABILITY_USER' ) && STATERA_MCP_OBSERVABILITY_USER ) {
			$login = (string) STATERA_MCP_OBSERVABILITY_USER;
		} else {
			$env   = getenv( 'WP_MCP_OBSERVABILITY_USER' );
			$login = is_string( $env ) ? trim( $env ) : '';
		}
		if ( '' === $login ) {
			return;
		}
		$user = get_user_by( 'login', $login );
		if ( ! $user || $user->has_cap( self::CAPABILITY ) ) {
			return;
		}
		$user->add_cap( self::CAPABILITY );
	}

	/* --------------------------------------------------------------- routes */

	public static function register_routes() {
		register_rest_route(
			self::REST_NS,
			'/debug',
			array(
				'methods'             => WP_REST_Server::READABLE,
				'callback'            => array( __CLASS__, 'route_get_debug' ),
				'permission_callback' => array( __CLASS__, 'authorize' ),
			)
		);

		register_rest_route(
			self::REST_NS,
			'/logs',
			array(
				'methods'             => WP_REST_Server::READABLE,
				'callback'            => array( __CLASS__, 'route_get_logs' ),
				'permission_callback' => array( __CLASS__, 'authorize' ),
				'args'                => array(
					'source' => array(
						'type'    => 'string',
						'default' => 'all',
						'enum'    => array( 'all', 'debug', 'woocommerce', 'cron', 'callbacks' ),
					),
					'level'  => array(
						'type' => 'string',
						'enum' => array( 'critical', 'error', 'warning', 'notice', 'info', 'debug' ),
					),
					'since'  => array( 'type' => 'string' ),
					'limit'  => array(
						'type'    => 'integer',
						'default' => self::LIMIT_DEFAULT,
						'minimum' => 1,
						'maximum' => self::LIMIT_MAX,
					),
					'search' => array( 'type' => 'string' ),
				),
			)
		);
	}

	public static function route_get_debug( $request ) {
		return rest_ensure_response( self::debug_state() );
	}

	/**
	 * Projecção só de leitura do estado de debug. Quem liga e desliga é o host
	 * (WORDPRESS_DEBUG no container); aqui apenas se diz o que ficou em vigor e
	 * se avisa quando a configuração não produz logs úteis.
	 */
	private static function debug_state() {
		$wp_debug     = defined( 'WP_DEBUG' ) && WP_DEBUG;
		$wp_debug_log = defined( 'WP_DEBUG_LOG' ) && WP_DEBUG_LOG;
		$wp_display   = defined( 'WP_DEBUG_DISPLAY' ) && WP_DEBUG_DISPLAY;

		$log_file = self::debug_log_file();
		if ( defined( 'WP_DEBUG_LOG' ) && is_string( WP_DEBUG_LOG ) && WP_DEBUG_LOG ) {
			$log_file = WP_DEBUG_LOG;
		} elseif ( $wp_debug_log ) {
			$log_file = WP_CONTENT_DIR . '/debug.log';
		}

		$notice = '';
		if ( $wp_debug && ! $wp_debug_log ) {
			$notice = 'WP_DEBUG está on mas WP_DEBUG_LOG está off: os erros não vão para ficheiro nenhum (só para o stderr do container). Definir WP_DEBUG_LOG via WORDPRESS_CONFIG_EXTRA.';
		} elseif ( ! $wp_debug ) {
			$notice = 'WP_DEBUG está off: o debug.log não recebe nada de novo. Ligar no host (env WORDPRESS_DEBUG do container).';
		}
		if ( $wp_display ) {
			$notice = trim( $notice . ' WP_DEBUG_DISPLAY está on: os erros estão a ser mostrados aos visitantes. Desligar via WORDPRESS_CONFIG_EXTRA.' );
		}

		return array(
			'wp_debug'         => $wp_debug,
			'wp_debug_log'     => $wp_debug_log,
			'wp_debug_display' => $wp_display,
			'log_path'         => self::relative_path( $log_file ),
			'log_size_bytes'   => file_exists( $log_file ) ? (int) filesize( $log_file ) : 0,
			'log_writable'     => is_dir( self::log_dir() ) && is_writable( self::log_dir() ),
			'plugin_version'   => self::VERSION,
			'notice'           => $notice,
		);
	}

	/* ------------------------------------------------------------- log reads */

	public static function route_get_logs( $request ) {
		$source = (string) $request->get_param( 'source' );
		$level  = (string) $request->get_param( 'level' );
		$search = (string) $request->get_param( 'search' );
		$limit  = (int) $request->get_param( 'limit' );
		$limit  = $limit > 0 ? min( $limit, self::LIMIT_MAX ) : self::LIMIT_DEFAULT;
		$since  = self::parse_since( (string) $request->get_param( 'since' ) );

		$sources = ( '' === $source || 'all' === $source )
			? array( 'debug', 'woocommerce', 'cron', 'callbacks' )
			: array( $source );

		$entries     = array();
		$read        = array();
		$unavailable = array();

		foreach ( $sources as $one ) {
			switch ( $one ) {
				case 'debug':
					$found = self::collect_debug();
					break;
				case 'woocommerce':
					$found = self::collect_woocommerce();
					break;
				case 'cron':
					$found = self::collect_cron();
					break;
				case 'callbacks':
					$found = self::collect_callbacks();
					break;
				default:
					$found = null;
			}
			if ( null === $found ) {
				$unavailable[] = $one;
				continue;
			}
			$read[]  = $one;
			$entries = array_merge( $entries, $found );
		}

		$entries = self::filter_entries( $entries, $level, $since, $search );
		usort( $entries, array( __CLASS__, 'sort_by_time_desc' ) );

		$truncated = count( $entries ) > $limit;
		if ( $truncated ) {
			$entries = array_slice( $entries, 0, $limit );
		}

		return rest_ensure_response(
			array(
				'source'       => ( '' === $source ? 'all' : $source ),
				'entries'      => array_values( $entries ),
				'truncated'    => $truncated,
				'sources_read' => $read,
				'unavailable'  => $unavailable,
			)
		);
	}

	private static function collect_debug() {
		$files = array();
		if ( defined( 'WP_DEBUG_LOG' ) && is_string( WP_DEBUG_LOG ) && WP_DEBUG_LOG ) {
			$files[] = WP_DEBUG_LOG;
		}
		$files[] = self::debug_log_file();
		$files[] = WP_CONTENT_DIR . '/debug.log';

		$seen    = array();
		$entries = array();
		foreach ( $files as $file ) {
			$real = @realpath( $file );
			if ( ! $real || isset( $seen[ $real ] ) || ! is_readable( $real ) ) {
				continue;
			}
			$seen[ $real ] = true;
			$entries       = array_merge( $entries, self::parse_php_log( self::tail( $real, self::DEBUG_TAIL_BYTES ), 'debug', 'php' ) );
		}
		if ( ! $seen ) {
			return array();
		}
		return $entries;
	}

	private static function collect_woocommerce() {
		$uploads = wp_upload_dir();
		if ( empty( $uploads['basedir'] ) ) {
			return null;
		}
		$dir = trailingslashit( $uploads['basedir'] ) . 'wc-logs';
		if ( ! is_dir( $dir ) ) {
			return null;
		}
		$files = glob( $dir . '/*.log' );
		if ( ! $files ) {
			return array();
		}
		usort(
			$files,
			static function ( $a, $b ) {
				return (int) @filemtime( $b ) <=> (int) @filemtime( $a );
			}
		);
		$files   = array_slice( $files, 0, self::WC_MAX_FILES );
		$entries = array();
		foreach ( $files as $file ) {
			if ( ! is_readable( $file ) ) {
				continue;
			}
			$channel = preg_replace( '/-\d{4}-\d{2}-\d{2}(-[a-f0-9]{8,})?\.log$/', '', basename( $file ) );
			$entries = array_merge( $entries, self::parse_wc_log( self::tail( $file, self::WC_TAIL_BYTES ), $channel ) );
		}
		return $entries;
	}

	private static function collect_cron() {
		$entries = array();
		$now     = time();

		$disabled  = defined( 'DISABLE_WP_CRON' ) && DISABLE_WP_CRON;
		$entries[] = self::entry(
			array(
				'source'    => 'cron',
				'channel'   => 'wp-cron',
				'level'     => $disabled ? 'warning' : 'info',
				'timestamp' => gmdate( 'c', $now ),
				'status'    => $disabled ? 'disabled' : 'enabled',
				'message'   => $disabled
					? 'DISABLE_WP_CRON está activo: os eventos só correm por cron do servidor.'
					: 'WP-Cron activo (disparado por tráfego do site).',
			)
		);

		$crons = function_exists( '_get_cron_array' ) ? _get_cron_array() : array();
		if ( is_array( $crons ) ) {
			foreach ( $crons as $timestamp => $hooks ) {
				if ( ! is_array( $hooks ) ) {
					continue;
				}
				foreach ( $hooks as $hook => $events ) {
					if ( ! is_array( $events ) ) {
						continue;
					}
					foreach ( $events as $event ) {
						$schedule = isset( $event['schedule'] ) && $event['schedule'] ? (string) $event['schedule'] : 'single';
						$late     = $now - (int) $timestamp;
						$overdue  = $late > 300;
						$entries[] = self::entry(
							array(
								'source'    => 'cron',
								'channel'   => (string) $hook,
								'level'     => $overdue ? 'warning' : 'info',
								'timestamp' => gmdate( 'c', (int) $timestamp ),
								'next_run'  => gmdate( 'c', (int) $timestamp ),
								'schedule'  => $schedule,
								'status'    => $overdue ? 'overdue' : 'scheduled',
								'overdue'   => $overdue,
								'message'   => $overdue
									? sprintf( 'Evento atrasado %d min (schedule: %s).', (int) floor( $late / 60 ), $schedule )
									: sprintf( 'Próxima execução em %d min (schedule: %s).', (int) ceil( max( 0, -$late ) / 60 ), $schedule ),
							)
						);
					}
				}
			}
		}

		return array_merge( $entries, self::collect_action_scheduler() );
	}

	private static function collect_action_scheduler() {
		if ( ! function_exists( 'as_get_scheduled_actions' ) ) {
			return array();
		}
		$entries = array();
		foreach ( array( 'failed', 'pending' ) as $status ) {
			try {
				$actions = as_get_scheduled_actions(
					array(
						'status'   => $status,
						'per_page' => 50,
						'orderby'  => 'date',
						'order'    => 'DESC',
					)
				);
			} catch ( Exception $e ) {
				continue;
			}
			if ( ! is_array( $actions ) ) {
				continue;
			}
			foreach ( $actions as $action ) {
				if ( ! is_object( $action ) || ! method_exists( $action, 'get_hook' ) ) {
					continue;
				}
				$when = '';
				if ( method_exists( $action, 'get_schedule' ) ) {
					$schedule = $action->get_schedule();
					if ( $schedule && method_exists( $schedule, 'get_date' ) ) {
						$date = $schedule->get_date();
						if ( $date instanceof DateTime ) {
							$when = $date->format( 'c' );
						}
					}
				}
				$entries[] = self::entry(
					array(
						'source'    => 'cron',
						'channel'   => 'action-scheduler:' . $action->get_hook(),
						'level'     => 'failed' === $status ? 'error' : 'info',
						'timestamp' => $when ? $when : gmdate( 'c' ),
						'next_run'  => $when,
						'status'    => $status,
						'message'   => sprintf( 'Action Scheduler: acção %s (%s).', $status, $action->get_hook() ),
					)
				);
			}
		}
		return $entries;
	}

	private static function collect_callbacks() {
		$file = self::callback_log_file();
		if ( ! file_exists( $file ) || ! is_readable( $file ) ) {
			return array();
		}
		$entries = array();
		$lines   = preg_split( "/\r\n|\n|\r/", self::tail( $file, self::WC_TAIL_BYTES ) );
		foreach ( $lines as $line ) {
			$line = trim( $line );
			if ( '' === $line ) {
				continue;
			}
			$data = json_decode( $line, true );
			if ( ! is_array( $data ) || empty( $data['message'] ) ) {
				continue;
			}
			$data['source'] = 'callbacks';
			$entries[]      = self::entry( $data );
		}
		return $entries;
	}

	/* -------------------------------------------------------------- parsing */

	private static function parse_php_log( $raw, $source, $channel ) {
		$entries = array();
		if ( '' === trim( (string) $raw ) ) {
			return $entries;
		}
		$lines = preg_split( "/\r\n|\n|\r/", $raw );
		$last  = null;
		foreach ( $lines as $line ) {
			if ( '' === trim( $line ) ) {
				continue;
			}
			if ( preg_match( '/^\[([^\]]+)\]\s?(.*)$/', $line, $m ) ) {
				$message   = $m[2];
				$entries[] = self::entry(
					array(
						'source'    => $source,
						'channel'   => $channel,
						'level'     => self::level_from_message( $message ),
						'timestamp' => self::to_iso8601( $m[1] ),
						'message'   => $message,
						'context'   => self::context_from_message( $message ),
					)
				);
				$last = count( $entries ) - 1;
				continue;
			}
			// Continuação (stack trace) junta-se à entrada anterior, com tecto.
			if ( null !== $last && isset( $entries[ $last ] ) ) {
				$entries[ $last ]['message'] = self::clip( $entries[ $last ]['message'] . ' | ' . trim( $line ) );
			}
		}
		return $entries;
	}

	private static function parse_wc_log( $raw, $channel ) {
		$entries = array();
		if ( '' === trim( (string) $raw ) ) {
			return $entries;
		}
		$lines = preg_split( "/\r\n|\n|\r/", $raw );
		$last  = null;
		foreach ( $lines as $line ) {
			if ( '' === trim( $line ) ) {
				continue;
			}
			if ( preg_match( '/^(\d{4}-\d{2}-\d{2}T[0-9:.+\-Z]+)\s+([A-Z]+)\s+(.*)$/', $line, $m ) ) {
				$entries[] = self::entry(
					array(
						'source'    => 'woocommerce',
						'channel'   => $channel,
						'level'     => strtolower( $m[2] ),
						'timestamp' => self::to_iso8601( $m[1] ),
						'message'   => $m[3],
					)
				);
				$last = count( $entries ) - 1;
				continue;
			}
			if ( null !== $last && isset( $entries[ $last ] ) ) {
				$entries[ $last ]['message'] = self::clip( $entries[ $last ]['message'] . ' | ' . trim( $line ) );
			}
		}
		return $entries;
	}

	private static function level_from_message( $message ) {
		if ( preg_match( '/fatal error|uncaught|parse error/i', $message ) ) {
			return 'critical';
		}
		if ( preg_match( '/\berror\b/i', $message ) ) {
			return 'error';
		}
		if ( preg_match( '/warning/i', $message ) ) {
			return 'warning';
		}
		if ( preg_match( '/deprecated|notice/i', $message ) ) {
			return 'notice';
		}
		return 'info';
	}

	private static function context_from_message( $message ) {
		if ( preg_match( '/ in (\/[^\s]+\.php) on line (\d+)/', $message, $m ) ) {
			return self::relative_path( $m[1] ) . ':' . $m[2];
		}
		return '';
	}

	/* --------------------------------------------------------- callback logs */

	public static function log_webhook_delivery( $http_args, $response, $duration, $arg, $webhook_id ) {
		$code = is_wp_error( $response ) ? 0 : (int) wp_remote_retrieve_response_code( $response );
		$msg  = is_wp_error( $response )
			? 'Entrega de webhook falhou: ' . $response->get_error_message()
			: sprintf( 'Entrega de webhook respondeu %d em %.2fs.', $code, (float) $duration );
		self::write_callback_log(
			array(
				'channel' => 'woocommerce-webhook:' . (int) $webhook_id,
				'level'   => ( is_wp_error( $response ) || $code >= 400 || 0 === $code ) ? 'error' : 'info',
				'status'  => (string) $code,
				'message' => $msg,
			)
		);
	}

	public static function log_http_failure( $response, $context, $class, $parsed_args, $url ) {
		if ( 'response' !== $context ) {
			return;
		}
		$code = is_wp_error( $response ) ? 0 : (int) wp_remote_retrieve_response_code( $response );
		if ( ! is_wp_error( $response ) && $code < 400 ) {
			return;
		}
		$msg = is_wp_error( $response )
			? 'Chamada de saída falhou: ' . $response->get_error_message()
			: sprintf( 'Chamada de saída respondeu %d.', $code );
		self::write_callback_log(
			array(
				'channel' => 'http-out',
				'level'   => 'error',
				'status'  => (string) $code,
				'message' => $msg . ' ' . self::safe_url( $url ),
			)
		);
	}

	public static function log_rest_error( $response, $handler, $request ) {
		if ( ! is_wp_error( $response ) || ! is_object( $request ) || ! method_exists( $request, 'get_route' ) ) {
			return $response;
		}
		$route = (string) $request->get_route();
		if ( ! preg_match( '/(webhook|callback|ipn|notify|payment|gateway)/i', $route ) ) {
			return $response;
		}
		self::write_callback_log(
			array(
				'channel' => 'rest-in',
				'level'   => 'error',
				'status'  => (string) $response->get_error_code(),
				'message' => sprintf( 'Callback recebido em %s falhou: %s', $route, $response->get_error_message() ),
			)
		);
		return $response;
	}

	/**
	 * Escreve uma linha JSON no log de callbacks. Nunca pode partir o pedido do site.
	 */
	private static function write_callback_log( $entry ) {
		try {
			if ( ! self::ensure_log_dir() ) {
				return;
			}
			$file = self::callback_log_file();
			if ( file_exists( $file ) && (int) @filesize( $file ) > self::CALLBACK_LOG_MAX ) {
				@rename( $file, $file . '.1' );
			}
			$line = wp_json_encode(
				array(
					'timestamp' => gmdate( 'c' ),
					'channel'   => isset( $entry['channel'] ) ? $entry['channel'] : 'callback',
					'level'     => isset( $entry['level'] ) ? $entry['level'] : 'info',
					'status'    => isset( $entry['status'] ) ? $entry['status'] : '',
					'message'   => self::clip( self::scrub( (string) $entry['message'] ) ),
				)
			);
			@file_put_contents( $file, $line . "\n", FILE_APPEND | LOCK_EX );
		} catch ( Throwable $e ) {
			return;
		}
	}

	/* --------------------------------------------------------------- helpers */

	private static function entry( $data ) {
		$entry = array(
			'source'    => isset( $data['source'] ) ? (string) $data['source'] : '',
			'channel'   => isset( $data['channel'] ) ? (string) $data['channel'] : '',
			'level'     => isset( $data['level'] ) ? (string) $data['level'] : 'info',
			'timestamp' => isset( $data['timestamp'] ) ? (string) $data['timestamp'] : '',
			'message'   => self::clip( self::scrub( isset( $data['message'] ) ? (string) $data['message'] : '' ) ),
		);
		foreach ( array( 'context', 'schedule', 'next_run', 'status' ) as $key ) {
			if ( ! empty( $data[ $key ] ) ) {
				$entry[ $key ] = self::scrub( (string) $data[ $key ] );
			}
		}
		if ( isset( $data['overdue'] ) ) {
			$entry['overdue'] = (bool) $data['overdue'];
		}
		return $entry;
	}

	/**
	 * Primeira passagem de sanitização (o MCP repete-a do lado de Go).
	 */
	private static function scrub( $text ) {
		if ( '' === $text ) {
			return $text;
		}
		$rules = array(
			'/[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}/'                                         => '[redacted:email]',
			'/\b(authorization|cookie|set-cookie|x-wp-nonce)\b\s*[:=]\s*(?:bearer\s+|basic\s+)?\S+/i'     => '$1: [redacted]',
			'/\b(api[_\-]?key|consumer_key|consumer_secret|client_secret|secret|token|password|pass)\b["\']?\s*[:=]\s*["\']?[^\s"\',;&)]+/i' => '$1=[redacted]',
			'/\b(ck|cs)_[a-f0-9]{20,}/i'                                                                  => '[redacted:wc-key]',
			'/\b\d(?:[ \-]?\d){12,18}\b/'                                                                 => '[redacted:pan]',
			'/\+\d[\d \-().]{7,}\d/'                                                                      => '[redacted:phone]',
			'/\b(?:\d{1,3}\.){3}\d{1,3}\b/'                                                               => '[redacted:ip]',
		);
		foreach ( $rules as $pattern => $replacement ) {
			$text = preg_replace( $pattern, $replacement, $text );
		}
		return trim( (string) $text );
	}

	private static function clip( $text ) {
		if ( function_exists( 'mb_strlen' ) && mb_strlen( $text ) > self::MESSAGE_MAX_CHARS ) {
			return mb_substr( $text, 0, self::MESSAGE_MAX_CHARS ) . ' […truncated]';
		}
		if ( strlen( $text ) > self::MESSAGE_MAX_CHARS * 2 ) {
			return substr( $text, 0, self::MESSAGE_MAX_CHARS ) . ' […truncated]';
		}
		return $text;
	}

	private static function safe_url( $url ) {
		$parts = wp_parse_url( (string) $url );
		if ( ! $parts || empty( $parts['host'] ) ) {
			return '';
		}
		$scheme = isset( $parts['scheme'] ) ? $parts['scheme'] . '://' : '';
		$path   = isset( $parts['path'] ) ? $parts['path'] : '';
		return $scheme . $parts['host'] . $path;
	}

	private static function relative_path( $path ) {
		$path = (string) $path;
		if ( defined( 'ABSPATH' ) && ABSPATH && 0 === strpos( $path, ABSPATH ) ) {
			return substr( $path, strlen( ABSPATH ) );
		}
		return basename( $path );
	}

	private static function to_iso8601( $raw ) {
		$ts = strtotime( (string) $raw );
		return $ts ? gmdate( 'c', $ts ) : '';
	}

	private static function parse_since( $raw ) {
		$raw = trim( $raw );
		if ( '' === $raw ) {
			return 0;
		}
		$ts = strtotime( $raw );
		return $ts ? (int) $ts : 0;
	}

	private static function filter_entries( $entries, $level, $since, $search ) {
		$out = array();
		foreach ( $entries as $entry ) {
			if ( '' !== $level && isset( $entry['level'] ) && $entry['level'] !== $level ) {
				continue;
			}
			if ( $since > 0 && ! empty( $entry['timestamp'] ) ) {
				$ts = strtotime( $entry['timestamp'] );
				if ( $ts && $ts < $since ) {
					continue;
				}
			}
			if ( '' !== $search && false === stripos( $entry['message'] . ' ' . $entry['channel'], $search ) ) {
				continue;
			}
			$out[] = $entry;
		}
		return $out;
	}

	public static function sort_by_time_desc( $a, $b ) {
		$ta = empty( $a['timestamp'] ) ? 0 : (int) strtotime( $a['timestamp'] );
		$tb = empty( $b['timestamp'] ) ? 0 : (int) strtotime( $b['timestamp'] );
		return $tb <=> $ta;
	}

	private static function tail( $file, $max_bytes ) {
		$size = @filesize( $file );
		$fh   = @fopen( $file, 'rb' );
		if ( ! $fh ) {
			return '';
		}
		if ( $size && $size > $max_bytes ) {
			@fseek( $fh, $size - $max_bytes );
			@fgets( $fh ); // descarta a primeira linha partida
		}
		$data = @stream_get_contents( $fh );
		@fclose( $fh );
		return is_string( $data ) ? $data : '';
	}
}

Statera_MCP_Observability::boot();
