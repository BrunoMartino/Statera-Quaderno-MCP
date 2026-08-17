package woocommerce

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"woocommerce-store-mcp/internal/guard"
)

type recordingAuth struct{}

func (recordingAuth) Apply(req *http.Request) {
	req.SetBasicAuth("mcp-content", "test-key-123")
}

func newWCClient(t *testing.T, h http.Handler) *Client {
	t.Helper()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	policy := guard.NewAccessPolicy(nil)
	caller := guard.NewStoreCaller(ts.URL, recordingAuth{}, policy)
	return NewClient(caller, policy)
}

func TestUpdateProductDescriptionDoesNotSendCommercialFields(t *testing.T) {
	var gotBody []byte
	client := newWCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.Write([]byte(`{"id":5,"name":"Mug","description":"Nice","regular_price":"99","stock_quantity":3,"sku":"SKU1","meta_data":[{"key":"x"}]}`))
	}))
	p, err := client.UpdateProductContent(context.Background(), 5, json.RawMessage(`{"description":"Nice"}`))
	if err != nil {
		t.Fatalf("%v", err)
	}
	if strings.Contains(string(gotBody), "regular_price") || strings.Contains(string(gotBody), "stock") {
		t.Fatalf("commercial fields in PATCH: %s", gotBody)
	}
	encoded, _ := json.Marshal(p)
	if strings.Contains(string(encoded), "regular_price") || strings.Contains(string(encoded), "stock_quantity") || strings.Contains(string(encoded), "sku") || strings.Contains(string(encoded), "meta_data") {
		t.Fatalf("unsanitized result: %s", encoded)
	}
}

func TestUpdateProductRegularPriceIsFieldForbiddenNoHTTP(t *testing.T) {
	var hits int
	client := newWCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
	}))
	_, err := client.UpdateProductContent(context.Background(), 5, json.RawMessage(`{"regular_price":"10"}`))
	if !guard.IsCode(err, guard.FieldForbidden) {
		t.Fatalf("got %v", err)
	}
	if hits != 0 {
		t.Fatalf("hits=%d", hits)
	}
}

func TestGetProductContentStripsPriceStockSKU(t *testing.T) {
	client := newWCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":5,"name":"Mug","slug":"mug","permalink":"https://loja.example.com/p/mug","description":"D","short_description":"S","regular_price":"50","sale_price":"40","price":"40","stock_quantity":2,"sku":"ABC","meta_data":[{"id":1,"key":"cost","value":"1"}],"email":"x@y.z"}`))
	}))
	p, err := client.GetProductContent(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(p)
	for _, bad := range []string{"regular_price", "sale_price", `"price"`, "stock_quantity", `"sku"`, "meta_data", "x@y.z"} {
		if strings.Contains(string(encoded), bad) {
			t.Fatalf("result contains %s: %s", bad, encoded)
		}
	}
	if p.Name != "Mug" || p.Description != "D" {
		t.Fatalf("editorial missing: %+v", p)
	}
}

func TestCreateCouponPercent10Posts(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte
	client := newWCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		w.Write([]byte(`{"id":8,"code":"SAVE10","discount_type":"percent","amount":"10","date_expires":"2026-12-31T00:00:00"}`))
	}))
	c, err := client.CreateCoupon(context.Background(), json.RawMessage(`{"code":"SAVE10","discount_type":"percent","amount":"10","date_expires":"2026-12-31T00:00:00","product_ids":[1]}`), false)
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/wp-json/wc/v3/coupons" {
		t.Fatalf("got %s %s", gotMethod, gotPath)
	}
	if c.Code != "SAVE10" || c.Amount != "10" {
		t.Fatalf("coupon=%+v", c)
	}
	if strings.Contains(string(gotBody), "human_confirmed") {
		t.Fatalf("tool flag leaked to WC: %s", gotBody)
	}
}

func TestCreateCouponPercentOver20WithoutConfirmNoHTTP(t *testing.T) {
	var hits int
	client := newWCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
	}))
	_, err := client.CreateCoupon(context.Background(), json.RawMessage(`{"code":"BIG","discount_type":"percent","amount":"25"}`), false)
	if !guard.IsCode(err, guard.DiscountConfirmationRequired) {
		t.Fatalf("got %v", err)
	}
	if hits != 0 {
		t.Fatalf("hits=%d", hits)
	}
}

func TestCreateCouponPercentOver20WithConfirmPosts(t *testing.T) {
	var hits int
	client := newWCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Write([]byte(`{"id":9,"code":"BIG","discount_type":"percent","amount":"25"}`))
	}))
	_, err := client.CreateCoupon(context.Background(), json.RawMessage(`{"code":"BIG","discount_type":"percent","amount":"25"}`), true)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("hits=%d", hits)
	}
}

func TestDeleteCouponAllowed(t *testing.T) {
	var gotMethod, gotPath string
	client := newWCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.Write([]byte(`{"id":8,"code":"SAVE10"}`))
	}))
	if _, err := client.DeleteCoupon(context.Background(), 8); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/wp-json/wc/v3/coupons/8" {
		t.Fatalf("got %s %s", gotMethod, gotPath)
	}
}

func TestCouponMetaDataIsFieldForbidden(t *testing.T) {
	var hits int
	client := newWCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
	}))
	_, err := client.CreateCoupon(context.Background(), json.RawMessage(`{"code":"X","discount_type":"fixed_cart","amount":"5","meta_data":[]}`), false)
	if !guard.IsCode(err, guard.FieldForbidden) {
		t.Fatalf("got %v", err)
	}
	if hits != 0 {
		t.Fatalf("hits=%d", hits)
	}
}
