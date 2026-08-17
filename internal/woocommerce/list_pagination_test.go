package woocommerce

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
)

func TestListProductsWalksAllPages(t *testing.T) {
	var pages []int
	client := newWCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/wp-json/wc/v3/products" {
			t.Fatalf("got %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("per_page") != "100" {
			t.Fatalf("per_page=%s", r.URL.Query().Get("per_page"))
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pages = append(pages, page)
		n := 100
		base := 1
		if page == 2 {
			n = 3
			base = 101
		}
		items := make([]ProductListItem, n)
		for i := 0; i < n; i++ {
			items[i] = ProductListItem{ID: base + i, Name: "P", Status: "publish"}
		}
		json.NewEncoder(w).Encode(items)
	}))
	got, err := client.ListProducts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 103 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].ID != 1 || got[102].ID != 103 {
		t.Fatalf("ids %d ... %d", got[0].ID, got[102].ID)
	}
	if len(pages) != 2 || pages[0] != 1 || pages[1] != 2 {
		t.Fatalf("pages=%v", pages)
	}
}

func TestListCouponsWalksAllPages(t *testing.T) {
	var pages []int
	client := newWCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wp-json/wc/v3/coupons" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("per_page") != "100" {
			t.Fatalf("per_page=%s", r.URL.Query().Get("per_page"))
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pages = append(pages, page)
		n := 100
		base := 1
		if page == 2 {
			n = 4
			base = 101
		}
		items := make([]Coupon, n)
		for i := 0; i < n; i++ {
			items[i] = Coupon{ID: base + i, Code: "c"}
		}
		json.NewEncoder(w).Encode(items)
	}))
	got, err := client.ListCoupons(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 104 {
		t.Fatalf("len=%d", len(got))
	}
	if len(pages) != 2 || pages[0] != 1 || pages[1] != 2 {
		t.Fatalf("pages=%v", pages)
	}
}
