package woocommerce

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"woocommerce-store-mcp/internal/guard"
)

const wcOrderFixture = `{
  "id": 727,
  "number": "727",
  "status": "processing",
  "currency": "USD",
  "total": "29.35",
  "date_paid": "2017-03-22T16:28:08",
  "date_completed": null,
  "payment_method": "bacs",
  "payment_method_title": "Direct Bank Transfer",
  "transaction_id": "",
  "billing": {
    "first_name": "John",
    "email": "john.doe@example.com",
    "phone": "(555) 555-5555",
    "address_1": "969 Market"
  },
  "shipping": {
    "first_name": "John",
    "address_1": "969 Market"
  },
  "refunds": [],
  "meta_data": [{"key": "_order_stock_reduced", "value": "yes"}]
}`

func TestListOrdersStripsPIIAndMapsRefunds(t *testing.T) {
	var gotStatus string
	client := newWCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/wp-json/wc/v3/orders" {
			t.Fatalf("got %s %s", r.Method, r.URL.Path)
		}
		gotStatus = r.URL.Query().Get("status")
		w.Write([]byte("[" + wcOrderFixture + "]"))
	}))
	got, err := client.ListOrders(context.Background(), "processing")
	if err != nil {
		t.Fatal(err)
	}
	if gotStatus != "processing" {
		t.Fatalf("status query=%q", gotStatus)
	}
	if len(got) != 1 || got[0].ID != 727 || got[0].RefundRequested {
		t.Fatalf("%+v", got)
	}
	raw, _ := json.Marshal(got)
	for _, bad := range []string{"john.doe@example.com", "(555) 555-5555", "969 Market", "billing", "shipping", "meta_data"} {
		if strings.Contains(string(raw), bad) {
			t.Fatalf("PII/extra field %q in %s", bad, raw)
		}
	}
}

func TestListOrdersRefundRequestedFromRefunds(t *testing.T) {
	client := newWCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":1,"status":"processing","refunds":[{"id":9,"total":"-10.00"}],"billing":{"email":"a@b.c"}}]`))
	}))
	got, err := client.ListOrders(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if !got[0].RefundRequested || len(got[0].Refunds) != 1 {
		t.Fatalf("%+v", got[0])
	}
}

func TestListOrdersInvalidStatusNoHTTP(t *testing.T) {
	var hits int
	client := newWCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
	}))
	_, err := client.ListOrders(context.Background(), "draft")
	if !guard.IsCode(err, guard.FieldForbidden) {
		t.Fatalf("got %v", err)
	}
	if hits != 0 {
		t.Fatalf("hits=%d", hits)
	}
}

func TestListPaymentsConfirmedFromDatePaid(t *testing.T) {
	client := newWCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[
			{"id":1,"payment_method":"bacs","payment_method_title":"BACS","transaction_id":"tx","date_paid":"2017-03-22T16:28:08","billing":{"email":"a@b.c"}},
			{"id":2,"payment_method":"cod","date_paid":null}
		]`))
	}))
	got, err := client.ListPayments(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !got[0].Confirmed || got[1].Confirmed {
		t.Fatalf("%+v", got)
	}
	raw, _ := json.Marshal(got)
	if strings.Contains(string(raw), "a@b.c") {
		t.Fatalf("email leaked: %s", raw)
	}
}

func TestListShipmentsWaitingWithoutTracking(t *testing.T) {
	client := newWCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":3,"status":"processing","date_completed":null,"meta_data":[]}]`))
	}))
	got, err := client.ListShipments(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got[0].FulfillmentState != "waiting" || got[0].TrackingCode != nil {
		t.Fatalf("%+v", got[0])
	}
}

func TestListShipmentsTrackingFromAllowlistedMeta(t *testing.T) {
	client := newWCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":4,"status":"processing","date_completed":null,"meta_data":[{"key":"_wc_shipment_tracking_items","value":[{"tracking_number":"DHL123"}]}]}]`))
	}))
	got, err := client.ListShipments(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got[0].TrackingCode == nil || *got[0].TrackingCode != "DHL123" || got[0].FulfillmentState != "shipped" {
		t.Fatalf("%+v", got[0])
	}
}

func TestListOrdersWriteNeverIssued(t *testing.T) {
	client := newWCClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("write issued: %s %s", r.Method, r.URL.Path)
		}
		w.Write([]byte(`[]`))
	}))
	if _, err := client.ListOrders(context.Background(), "completed"); err != nil {
		t.Fatal(err)
	}
}
