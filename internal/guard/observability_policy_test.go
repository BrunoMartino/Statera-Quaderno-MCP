package guard

import "testing"

func TestObservabilityRoutesAllowlist(t *testing.T) {
	p := NewAccessPolicy(nil)
	allowed := []struct{ method, path string }{
		{"GET", "/wp-json/statera-mcp/v1/debug"},
		{"GET", "/wp-json/statera-mcp/v1/logs?source=debug&limit=10"},
	}
	for _, c := range allowed {
		if err := p.AssertRequest(c.method, c.path); err != nil {
			t.Fatalf("%s %s: %v", c.method, c.path, err)
		}
	}

	forbiddenMethod := []struct{ method, path string }{
		{"POST", "/wp-json/statera-mcp/v1/debug"},
		{"PUT", "/wp-json/statera-mcp/v1/debug"},
		{"DELETE", "/wp-json/statera-mcp/v1/debug"},
		{"PATCH", "/wp-json/statera-mcp/v1/debug"},
		{"POST", "/wp-json/statera-mcp/v1/logs"},
		{"DELETE", "/wp-json/statera-mcp/v1/logs"},
	}
	for _, c := range forbiddenMethod {
		if err := p.AssertRequest(c.method, c.path); !IsCode(err, MethodForbidden) {
			t.Fatalf("%s %s: want METHOD_FORBIDDEN, got %v", c.method, c.path, err)
		}
	}

	forbiddenRoute := []string{
		"/wp-json/statera-mcp/v1/logs/delete",
		"/wp-json/statera-mcp/v1/settings",
		"/wp-json/statera-mcp/v2/debug",
		"/wp-json/wp/v2/users",
	}
	for _, path := range forbiddenRoute {
		if err := p.AssertRequest("GET", path); !IsCode(err, RouteForbidden) {
			t.Fatalf("GET %s: want ROUTE_FORBIDDEN, got %v", path, err)
		}
	}
}

// O debug liga-se no host (WORDPRESS_DEBUG do container); este MCP só lê.
func TestDebugStateIsReadOnly(t *testing.T) {
	p := NewAccessPolicy(nil)
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		if err := p.AssertRequest(method, "/wp-json/statera-mcp/v1/debug"); !IsCode(err, MethodForbidden) {
			t.Fatalf("%s /debug: want METHOD_FORBIDDEN, got %v", method, err)
		}
	}
	if err := p.AssertWriteKeys(ResourceGet, []string{"enabled"}); !IsCode(err, FieldForbidden) {
		t.Fatalf("enabled key: want FIELD_FORBIDDEN, got %v", err)
	}
}

func TestLogQueryAllowlist(t *testing.T) {
	p := NewAccessPolicy(nil)
	if err := p.AssertWriteKeys(ResourceLogs, []string{"source", "level", "since", "limit", "search"}); err != nil {
		t.Fatalf("allowlisted log keys: %v", err)
	}
	if err := p.AssertWriteKeys(ResourceLogs, []string{"path"}); !IsCode(err, FieldForbidden) {
		t.Fatalf("path key: want FIELD_FORBIDDEN, got %v", err)
	}
	for _, s := range []string{"", "all", "debug", "woocommerce", "cron", "callbacks"} {
		if err := p.AssertLogSource(s); err != nil {
			t.Fatalf("source %q: %v", s, err)
		}
	}
	for _, s := range []string{"wp-config", "users", "database", "../../etc/passwd"} {
		if err := p.AssertLogSource(s); !IsCode(err, FieldForbidden) {
			t.Fatalf("source %q: want FIELD_FORBIDDEN, got %v", s, err)
		}
	}
	if err := p.AssertLogLevel("banana"); !IsCode(err, FieldForbidden) {
		t.Fatalf("level banana: want FIELD_FORBIDDEN, got %v", err)
	}
	if n, err := p.AssertLogLimit(0); err != nil || n != LogsLimitDefault {
		t.Fatalf("default limit: got %d %v", n, err)
	}
	if _, err := p.AssertLogLimit(LogsLimitMax + 1); !IsCode(err, FieldForbidden) {
		t.Fatalf("limit over max: want FIELD_FORBIDDEN, got %v", err)
	}
	if _, err := p.AssertLogLimit(-1); !IsCode(err, FieldForbidden) {
		t.Fatalf("negative limit: want FIELD_FORBIDDEN, got %v", err)
	}
}
