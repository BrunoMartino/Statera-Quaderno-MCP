package woocommerce

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"woocommerce-store-mcp/internal/guard"
)

func TestCreateCouponAmountOver20WithoutTypeNoHTTP(t *testing.T) {
	var hits int
	client := newWCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
	}))
	_, err := client.CreateCoupon(context.Background(), json.RawMessage(`{"code":"BIG","amount":"25"}`), false)
	if !guard.IsCode(err, guard.DiscountConfirmationRequired) {
		t.Fatalf("got %v", err)
	}
	if hits != 0 {
		t.Fatalf("hits=%d", hits)
	}
}

func TestUpdateCouponAmountOver20WithoutTypeNoPatch(t *testing.T) {
	var methods []string
	client := newWCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.Method == http.MethodGet {
			w.Write([]byte(`{"id":8,"code":"SAVE","discount_type":"percent","amount":"10"}`))
			return
		}
		t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
	}))
	_, err := client.UpdateCoupon(context.Background(), 8, json.RawMessage(`{"id":8,"amount":"25"}`), false)
	if !guard.IsCode(err, guard.DiscountConfirmationRequired) {
		t.Fatalf("got %v", err)
	}
	if len(methods) != 1 || methods[0] != http.MethodGet {
		t.Fatalf("methods=%v", methods)
	}
}

func TestCreateCouponAmountAsNumber(t *testing.T) {
	client := newWCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.Write([]byte(`{"id":3,"code":"N","discount_type":"percent","amount":"5"}`))
	}))
	c, err := client.CreateCoupon(context.Background(), json.RawMessage(`{"code":"N","discount_type":"percent","amount":5}`), false)
	if err != nil {
		t.Fatal(err)
	}
	if c.ID != 3 {
		t.Fatalf("id=%d", c.ID)
	}
}
