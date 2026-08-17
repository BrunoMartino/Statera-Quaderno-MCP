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

const (
	listPageSize = 100
	listMaxPages = 100
)

func (c *Client) ListProducts(ctx context.Context) ([]ProductListItem, error) {
	var all []ProductListItem
	for page := 1; page <= listMaxPages; page++ {
		path := "/wp-json/wc/v3/products?per_page=" + strconv.Itoa(listPageSize) + "&page=" + strconv.Itoa(page)
		b, err := c.caller.Do(ctx, http.MethodGet, path, nil, "")
		if err != nil {
			return nil, err
		}
		var items []ProductListItem
		if err := json.Unmarshal(b, &items); err != nil {
			return nil, err
		}
		if len(items) == 0 {
			return all, nil
		}
		all = append(all, items...)
		if len(items) < listPageSize {
			return all, nil
		}
	}
	return all, nil
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
	var all []Coupon
	for page := 1; page <= listMaxPages; page++ {
		path := "/wp-json/wc/v3/coupons?per_page=" + strconv.Itoa(listPageSize) + "&page=" + strconv.Itoa(page)
		b, err := c.caller.Do(ctx, http.MethodGet, path, nil, "")
		if err != nil {
			return nil, err
		}
		var items []Coupon
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
		all = []Coupon{}
	}
	return all, nil
}

func (c *Client) CreateCoupon(ctx context.Context, raw json.RawMessage, confirmed bool) (*Coupon, error) {
	body, err := c.prepareCoupon(ctx, 0, raw, confirmed)
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
	body, err := c.prepareCoupon(ctx, id, raw, confirmed)
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

func (c *Client) getCoupon(ctx context.Context, id int) (*Coupon, error) {
	b, err := c.caller.Do(ctx, http.MethodGet, "/wp-json/wc/v3/coupons/"+strconv.Itoa(id), nil, "")
	if err != nil {
		return nil, err
	}
	var coupon Coupon
	if err := json.Unmarshal(b, &coupon); err != nil {
		return nil, err
	}
	return &coupon, nil
}

func (c *Client) prepareCoupon(ctx context.Context, id int, raw json.RawMessage, confirmed bool) ([]byte, error) {
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
	discountType := ""
	if aux.DiscountType != nil {
		discountType = *aux.DiscountType
		if err := c.policy.AssertDiscountType(discountType); err != nil {
			return nil, err
		}
	}
	amount := ""
	if aux.Amount != nil {
		amount = string(*aux.Amount)
	}
	if amount != "" && discountType == "" {
		if id > 0 {
			existing, err := c.getCoupon(ctx, id)
			if err != nil {
				return nil, err
			}
			discountType = existing.DiscountType
		} else {
			discountType = "percent"
		}
	}
	if amount != "" {
		if err := c.policy.AssertCouponDiscount(discountType, amount, confirmed); err != nil {
			return nil, err
		}
	}
	var write CouponWrite
	if err := json.Unmarshal(raw, &write); err != nil {
		return nil, err
	}
	return json.Marshal(write)
}
