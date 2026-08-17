package woocommerce

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"woocommerce-store-mcp/internal/guard"
)

type Client struct {
	caller *guard.StoreCaller
	policy *guard.AccessPolicy
}

func NewClient(caller *guard.StoreCaller, policy *guard.AccessPolicy) *Client {
	return &Client{caller: caller, policy: policy}
}

func (c *Client) ListProducts(ctx context.Context) ([]ProductListItem, error) {
	b, err := c.caller.Do(ctx, http.MethodGet, "/wp-json/wc/v3/products", nil, "")
	if err != nil {
		return nil, err
	}
	var items []ProductListItem
	if err := json.Unmarshal(b, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *Client) GetProductContent(ctx context.Context, id int) (*ProductContent, error) {
	b, err := c.caller.Do(ctx, http.MethodGet, "/wp-json/wc/v3/products/"+strconv.Itoa(id), nil, "")
	if err != nil {
		return nil, err
	}
	var p ProductContent
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *Client) UpdateProductContent(ctx context.Context, id int, raw json.RawMessage) (*ProductContent, error) {
	keys, err := guard.JSONObjectKeys(raw)
	if err != nil {
		return nil, err
	}
	if err := c.policy.AssertWriteKeys(guard.ResourceProduct, keys); err != nil {
		return nil, err
	}
	var write ProductWrite
	if err := json.Unmarshal(raw, &write); err != nil {
		return nil, err
	}
	if write.CatalogVisibility != nil {
		if err := c.policy.AssertCatalogVisibility(*write.CatalogVisibility); err != nil {
			return nil, err
		}
	}
	body, err := json.Marshal(write)
	if err != nil {
		return nil, err
	}
	b, err := c.caller.Do(ctx, http.MethodPatch, "/wp-json/wc/v3/products/"+strconv.Itoa(id), body, "application/json")
	if err != nil {
		return nil, err
	}
	var p ProductContent
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *Client) ListCoupons(ctx context.Context) ([]Coupon, error) {
	b, err := c.caller.Do(ctx, http.MethodGet, "/wp-json/wc/v3/coupons", nil, "")
	if err != nil {
		return nil, err
	}
	var items []Coupon
	if err := json.Unmarshal(b, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *Client) CreateCoupon(ctx context.Context, raw json.RawMessage, confirmed bool) (*Coupon, error) {
	body, err := c.prepareCoupon(raw, confirmed)
	if err != nil {
		return nil, err
	}
	b, err := c.caller.Do(ctx, http.MethodPost, "/wp-json/wc/v3/coupons", body, "application/json")
	if err != nil {
		return nil, err
	}
	var coupon Coupon
	if err := json.Unmarshal(b, &coupon); err != nil {
		return nil, err
	}
	return &coupon, nil
}

func (c *Client) UpdateCoupon(ctx context.Context, id int, raw json.RawMessage, confirmed bool) (*Coupon, error) {
	body, err := c.prepareCoupon(raw, confirmed)
	if err != nil {
		return nil, err
	}
	b, err := c.caller.Do(ctx, http.MethodPatch, "/wp-json/wc/v3/coupons/"+strconv.Itoa(id), body, "application/json")
	if err != nil {
		return nil, err
	}
	var coupon Coupon
	if err := json.Unmarshal(b, &coupon); err != nil {
		return nil, err
	}
	return &coupon, nil
}

func (c *Client) DeleteCoupon(ctx context.Context, id int) (*Coupon, error) {
	b, err := c.caller.Do(ctx, http.MethodDelete, "/wp-json/wc/v3/coupons/"+strconv.Itoa(id)+"?force=true", nil, "")
	if err != nil {
		return nil, err
	}
	var coupon Coupon
	if len(b) == 0 {
		return &Coupon{ID: id}, nil
	}
	if err := json.Unmarshal(b, &coupon); err != nil {
		return nil, err
	}
	return &coupon, nil
}

func (c *Client) prepareCoupon(raw json.RawMessage, confirmed bool) ([]byte, error) {
	keys, err := guard.JSONObjectKeys(raw)
	if err != nil {
		return nil, err
	}
	if err := c.policy.AssertWriteKeys(guard.ResourceCoupon, keys); err != nil {
		return nil, err
	}
	var aux struct {
		DiscountType *string       `json:"discount_type"`
		Amount       *couponAmount `json:"amount"`
	}
	if err := json.Unmarshal(raw, &aux); err != nil {
		return nil, err
	}
	discountType, amount := "", ""
	if aux.DiscountType != nil {
		discountType = *aux.DiscountType
		if err := c.policy.AssertDiscountType(discountType); err != nil {
			return nil, err
		}
	}
	if aux.Amount != nil {
		amount = string(*aux.Amount)
	}
	if err := c.policy.AssertCouponDiscount(discountType, amount, confirmed); err != nil {
		return nil, err
	}
	var write CouponWrite
	if err := json.Unmarshal(raw, &write); err != nil {
		return nil, err
	}
	return json.Marshal(write)
}
