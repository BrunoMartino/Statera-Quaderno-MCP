package guard

import "testing"

func TestAssertRequestUsersIsRouteForbidden(t *testing.T) {
	p := NewAccessPolicy(nil)
	err := p.AssertRequest("GET", "/wp-json/wp/v2/users")
	if !IsCode(err, RouteForbidden) {
		t.Fatalf("got %v, want ROUTE_FORBIDDEN", err)
	}
}

func TestAssertRequestDeletePostIsMethodForbidden(t *testing.T) {
	p := NewAccessPolicy(nil)
	err := p.AssertRequest("DELETE", "/wp-json/wp/v2/posts/1")
	if !IsCode(err, MethodForbidden) {
		t.Fatalf("got %v, want METHOD_FORBIDDEN", err)
	}
}

func TestAssertRequestDeleteCouponAllowed(t *testing.T) {
	p := NewAccessPolicy(nil)
	if err := p.AssertRequest("DELETE", "/wp-json/wc/v3/coupons/9"); err != nil {
		t.Fatalf("DELETE coupon: %v", err)
	}
}

func TestAssertRequestPostProductIsMethodForbidden(t *testing.T) {
	p := NewAccessPolicy(nil)
	err := p.AssertRequest("POST", "/wp-json/wc/v3/products")
	if !IsCode(err, MethodForbidden) {
		t.Fatalf("got %v, want METHOD_FORBIDDEN", err)
	}
}

func TestAssertRequestPutProductIsMethodForbidden(t *testing.T) {
	p := NewAccessPolicy(nil)
	err := p.AssertRequest("PUT", "/wp-json/wc/v3/products/1")
	if !IsCode(err, MethodForbidden) {
		t.Fatalf("got %v, want METHOD_FORBIDDEN", err)
	}
}

func TestAssertRequestVariationsIsRouteForbidden(t *testing.T) {
	p := NewAccessPolicy(nil)
	err := p.AssertRequest("PATCH", "/wp-json/wc/v3/products/1/variations")
	if !IsCode(err, RouteForbidden) {
		t.Fatalf("got %v, want ROUTE_FORBIDDEN", err)
	}
}

func TestAssertRequestDeleteMediaIsMethodForbidden(t *testing.T) {
	p := NewAccessPolicy(nil)
	err := p.AssertRequest("DELETE", "/wp-json/wp/v2/media/3")
	if !IsCode(err, MethodForbidden) {
		t.Fatalf("got %v, want METHOD_FORBIDDEN", err)
	}
}

func TestAssertWriteKeysRegularPriceIsFieldForbidden(t *testing.T) {
	p := NewAccessPolicy(nil)
	err := p.AssertWriteKeys(ResourceProduct, []string{"description", "regular_price"})
	if !IsCode(err, FieldForbidden) {
		t.Fatalf("got %v, want FIELD_FORBIDDEN", err)
	}
}

func TestAssertWriteKeysPostPasswordIsFieldForbidden(t *testing.T) {
	p := NewAccessPolicy(nil)
	err := p.AssertWriteKeys(ResourcePost, []string{"title", "password"})
	if !IsCode(err, FieldForbidden) {
		t.Fatalf("got %v, want FIELD_FORBIDDEN", err)
	}
}

func TestAssertWriteKeysPostAllowlistOK(t *testing.T) {
	p := NewAccessPolicy(nil)
	if err := p.AssertWriteKeys(ResourcePost, []string{"title", "content", "status"}); err != nil {
		t.Fatalf("%v", err)
	}
}

func TestAssertStatusPrivateIsFieldForbidden(t *testing.T) {
	p := NewAccessPolicy(nil)
	err := p.AssertStatus("private")
	if !IsCode(err, FieldForbidden) {
		t.Fatalf("got %v, want FIELD_FORBIDDEN", err)
	}
}

func TestAssertCouponPercentOver20WithoutConfirm(t *testing.T) {
	p := NewAccessPolicy(nil)
	err := p.AssertCouponDiscount("percent", "21", false)
	if !IsCode(err, DiscountConfirmationRequired) {
		t.Fatalf("got %v, want DISCOUNT_CONFIRMATION_REQUIRED", err)
	}
}

func TestAssertCouponPercent20Allowed(t *testing.T) {
	p := NewAccessPolicy(nil)
	if err := p.AssertCouponDiscount("percent", "20", false); err != nil {
		t.Fatalf("boundary 20: %v", err)
	}
}

func TestAssertCouponPercentOver20WithConfirm(t *testing.T) {
	p := NewAccessPolicy(nil)
	if err := p.AssertCouponDiscount("percent", "25", true); err != nil {
		t.Fatalf("confirmed: %v", err)
	}
}

func TestAssertCouponFixedCartNoConfirmNeeded(t *testing.T) {
	p := NewAccessPolicy(nil)
	if err := p.AssertCouponDiscount("fixed_cart", "100", false); err != nil {
		t.Fatalf("fixed_cart: %v", err)
	}
}

func TestJSONObjectKeysDetectsForbidden(t *testing.T) {
	keys, err := JSONObjectKeys([]byte(`{"description":"x","regular_price":"10"}`))
	if err != nil {
		t.Fatal(err)
	}
	p := NewAccessPolicy(nil)
	if err := p.AssertWriteKeys(ResourceProduct, keys); !IsCode(err, FieldForbidden) {
		t.Fatalf("got %v", err)
	}
}
