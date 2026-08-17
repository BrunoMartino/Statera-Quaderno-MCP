package woocommerce

type ProductListItem struct {
	ID        int    `json:"id"`
	Name      string `json:"name,omitempty"`
	Slug      string `json:"slug,omitempty"`
	Status    string `json:"status,omitempty"`
	Permalink string `json:"permalink,omitempty"`
}

type ProductImage struct {
	ID  int    `json:"id,omitempty"`
	Src string `json:"src,omitempty"`
	Alt string `json:"alt,omitempty"`
}

type ProductTerm struct {
	ID   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Slug string `json:"slug,omitempty"`
}

type ProductContent struct {
	ID                int            `json:"id"`
	Name              string         `json:"name,omitempty"`
	Slug              string         `json:"slug,omitempty"`
	Status            string         `json:"status,omitempty"`
	Permalink         string         `json:"permalink,omitempty"`
	Description       string         `json:"description,omitempty"`
	ShortDescription  string         `json:"short_description,omitempty"`
	Images            []ProductImage `json:"images,omitempty"`
	CatalogVisibility string         `json:"catalog_visibility,omitempty"`
	Categories        []ProductTerm  `json:"categories,omitempty"`
	Tags              []ProductTerm  `json:"tags,omitempty"`
}

type ProductWrite struct {
	Name              *string        `json:"name,omitempty"`
	Slug              *string        `json:"slug,omitempty"`
	Description       *string        `json:"description,omitempty"`
	ShortDescription  *string        `json:"short_description,omitempty"`
	Images            []ProductImage `json:"images,omitempty"`
	CatalogVisibility *string        `json:"catalog_visibility,omitempty"`
	Categories        []ProductTerm  `json:"categories,omitempty"`
	Tags              []ProductTerm  `json:"tags,omitempty"`
}

type Coupon struct {
	ID                int    `json:"id,omitempty"`
	Code              string `json:"code,omitempty"`
	DiscountType      string `json:"discount_type,omitempty"`
	Amount            string `json:"amount,omitempty"`
	DateExpires       string `json:"date_expires,omitempty"`
	DateExpiresGMT    string `json:"date_expires_gmt,omitempty"`
	ProductIDs        []int  `json:"product_ids,omitempty"`
	ProductCategories []int  `json:"product_categories,omitempty"`
}

type CouponWrite struct {
	Code              *string       `json:"code,omitempty"`
	DiscountType      *string       `json:"discount_type,omitempty"`
	Amount            *couponAmount `json:"amount,omitempty"`
	DateExpires       *string       `json:"date_expires,omitempty"`
	DateExpiresGMT    *string       `json:"date_expires_gmt,omitempty"`
	ProductIDs        []int         `json:"product_ids,omitempty"`
	ProductCategories []int         `json:"product_categories,omitempty"`
}

type OrderRefund struct {
	ID     int    `json:"id,omitempty"`
	Total  string `json:"total,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type OrderListItem struct {
	ID              int           `json:"id"`
	Number          string        `json:"number,omitempty"`
	Status          string        `json:"status,omitempty"`
	Currency        string        `json:"currency,omitempty"`
	Total           string        `json:"total,omitempty"`
	DatePaid        *string       `json:"date_paid"`
	RefundRequested bool          `json:"refund_requested"`
	Refunds         []OrderRefund `json:"refunds"`
}

type PaymentListItem struct {
	OrderID            int     `json:"order_id"`
	PaymentMethod      string  `json:"payment_method,omitempty"`
	PaymentMethodTitle string  `json:"payment_method_title,omitempty"`
	TransactionID      string  `json:"transaction_id"`
	DatePaid           *string `json:"date_paid"`
	Confirmed          bool    `json:"confirmed"`
}

type ShipmentListItem struct {
	OrderID          int     `json:"order_id"`
	FulfillmentState string  `json:"fulfillment_state"`
	TrackingCode     *string `json:"tracking_code"`
	DateCompleted    *string `json:"date_completed"`
}
