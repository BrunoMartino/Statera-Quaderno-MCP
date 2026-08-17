package woocommerce

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

type wcOrder struct {
	ID                 int           `json:"id"`
	Number             string        `json:"number"`
	Status             string        `json:"status"`
	Currency           string        `json:"currency"`
	Total              string        `json:"total"`
	DatePaid           *string       `json:"date_paid"`
	DateCompleted      *string       `json:"date_completed"`
	PaymentMethod      string        `json:"payment_method"`
	PaymentMethodTitle string        `json:"payment_method_title"`
	TransactionID      string        `json:"transaction_id"`
	Refunds            []OrderRefund `json:"refunds"`
	MetaData           []wcMeta      `json:"meta_data"`
}

type wcMeta struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

func (c *Client) ListOrders(ctx context.Context, status string) ([]OrderListItem, error) {
	if err := c.policy.AssertOrderStatus(status); err != nil {
		return nil, err
	}
	raw, err := c.listWCOrders(ctx, status)
	if err != nil {
		return nil, err
	}
	out := make([]OrderListItem, 0, len(raw))
	for _, o := range raw {
		out = append(out, projectOrder(o))
	}
	return out, nil
}

func (c *Client) ListPayments(ctx context.Context) ([]PaymentListItem, error) {
	raw, err := c.listWCOrders(ctx, "")
	if err != nil {
		return nil, err
	}
	out := make([]PaymentListItem, 0, len(raw))
	for _, o := range raw {
		out = append(out, projectPayment(o))
	}
	return out, nil
}

func (c *Client) ListShipments(ctx context.Context) ([]ShipmentListItem, error) {
	raw, err := c.listWCOrders(ctx, "")
	if err != nil {
		return nil, err
	}
	out := make([]ShipmentListItem, 0, len(raw))
	for _, o := range raw {
		out = append(out, projectShipment(o))
	}
	return out, nil
}

func (c *Client) listWCOrders(ctx context.Context, status string) ([]wcOrder, error) {
	var all []wcOrder
	for page := 1; page <= listMaxPages; page++ {
		q := url.Values{}
		q.Set("per_page", strconv.Itoa(listPageSize))
		q.Set("page", strconv.Itoa(page))
		if status != "" {
			q.Set("status", status)
		}
		path := "/wp-json/wc/v3/orders?" + q.Encode()
		b, err := c.caller.Do(ctx, http.MethodGet, path, nil, "")
		if err != nil {
			return nil, err
		}
		var items []wcOrder
		if err := json.Unmarshal(b, &items); err != nil {
			return nil, err
		}
		if len(items) == 0 {
			break
		}
		all = append(all, items...)
		if len(items) < listPageSize {
			break
		}
	}
	if all == nil {
		all = []wcOrder{}
	}
	return all, nil
}

func projectOrder(o wcOrder) OrderListItem {
	refunds := o.Refunds
	if refunds == nil {
		refunds = []OrderRefund{}
	}
	return OrderListItem{
		ID:              o.ID,
		Number:          o.Number,
		Status:          o.Status,
		Currency:        o.Currency,
		Total:           o.Total,
		DatePaid:        nonemptyDate(o.DatePaid),
		RefundRequested: o.Status == "refunded" || len(refunds) > 0,
		Refunds:         refunds,
	}
}

func projectPayment(o wcOrder) PaymentListItem {
	paid := nonemptyDate(o.DatePaid)
	return PaymentListItem{
		OrderID:            o.ID,
		PaymentMethod:      o.PaymentMethod,
		PaymentMethodTitle: o.PaymentMethodTitle,
		TransactionID:      o.TransactionID,
		DatePaid:           paid,
		Confirmed:          paid != nil,
	}
}

func projectShipment(o wcOrder) ShipmentListItem {
	completed := nonemptyDate(o.DateCompleted)
	tracking := trackingCode(o.MetaData)
	state := "waiting"
	if completed != nil || tracking != nil || o.Status == "completed" {
		state = "shipped"
	}
	return ShipmentListItem{
		OrderID:          o.ID,
		FulfillmentState: state,
		TrackingCode:     tracking,
		DateCompleted:    completed,
	}
}

func nonemptyDate(v *string) *string {
	if v == nil || *v == "" {
		return nil
	}
	return v
}

func trackingCode(meta []wcMeta) *string {
	for _, m := range meta {
		switch m.Key {
		case "_tracking_number", "tracking_number", "_tracking_code":
			var s string
			if json.Unmarshal(m.Value, &s) == nil && s != "" {
				return &s
			}
		case "_wc_shipment_tracking_items":
			var items []struct {
				TrackingNumber string `json:"tracking_number"`
			}
			if json.Unmarshal(m.Value, &items) == nil {
				for _, it := range items {
					if it.TrackingNumber != "" {
						n := it.TrackingNumber
						return &n
					}
				}
			}
		}
	}
	return nil
}
