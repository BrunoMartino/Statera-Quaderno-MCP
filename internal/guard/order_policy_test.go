package guard

import "testing"

func TestAssertRequestGetOrdersAllowed(t *testing.T) {
	p := NewAccessPolicy(nil)
	if err := p.AssertRequest("GET", "/wp-json/wc/v3/orders"); err != nil {
		t.Fatalf("GET orders: %v", err)
	}
	if err := p.AssertRequest("GET", "/wp-json/wc/v3/orders/727"); err != nil {
		t.Fatalf("GET order id: %v", err)
	}
	if err := p.AssertRequest("GET", "/wp-json/wc/v3/refunds"); err != nil {
		t.Fatalf("GET refunds: %v", err)
	}
}

func TestAssertRequestOrderWritesAreMethodForbidden(t *testing.T) {
	p := NewAccessPolicy(nil)
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		err := p.AssertRequest(method, "/wp-json/wc/v3/orders")
		if !IsCode(err, MethodForbidden) {
			t.Fatalf("%s collection: got %v, want METHOD_FORBIDDEN", method, err)
		}
		err = p.AssertRequest(method, "/wp-json/wc/v3/orders/1")
		if !IsCode(err, MethodForbidden) {
			t.Fatalf("%s id: got %v, want METHOD_FORBIDDEN", method, err)
		}
	}
}

func TestAssertRequestPaymentGatewaysAndCustomersAreRouteForbidden(t *testing.T) {
	p := NewAccessPolicy(nil)
	err := p.AssertRequest("GET", "/wp-json/wc/v3/payment_gateways")
	if !IsCode(err, RouteForbidden) {
		t.Fatalf("payment_gateways: got %v, want ROUTE_FORBIDDEN", err)
	}
	err = p.AssertRequest("GET", "/wp-json/wc/v3/customers")
	if !IsCode(err, RouteForbidden) {
		t.Fatalf("customers: got %v, want ROUTE_FORBIDDEN", err)
	}
	err = p.AssertRequest("GET", "/wp-json/wc/v3/shipping/zones")
	if !IsCode(err, RouteForbidden) {
		t.Fatalf("shipping/zones: got %v, want ROUTE_FORBIDDEN", err)
	}
}

func TestAssertOrderStatusAllowlist(t *testing.T) {
	p := NewAccessPolicy(nil)
	if err := p.AssertOrderStatus(""); err != nil {
		t.Fatalf("empty: %v", err)
	}
	if err := p.AssertOrderStatus("processing"); err != nil {
		t.Fatalf("processing: %v", err)
	}
	err := p.AssertOrderStatus("draft")
	if !IsCode(err, FieldForbidden) {
		t.Fatalf("draft: got %v, want FIELD_FORBIDDEN", err)
	}
}

func TestAssertWriteKeysOrderListStatusOK(t *testing.T) {
	p := NewAccessPolicy(nil)
	if err := p.AssertWriteKeys(ResourceOrderList, []string{"status"}); err != nil {
		t.Fatalf("%v", err)
	}
	err := p.AssertWriteKeys(ResourceOrderList, []string{"status", "email"})
	if !IsCode(err, FieldForbidden) {
		t.Fatalf("got %v, want FIELD_FORBIDDEN", err)
	}
}
