package guard

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type Resource string

const (
	ResourcePost    Resource = "post"
	ResourcePage    Resource = "page"
	ResourceProduct Resource = "product"
	ResourceCoupon  Resource = "coupon"
	ResourceMedia   Resource = "media"
	ResourceGet     Resource = "get"
	ResourceList    Resource = "list"
)

// AccessPolicy is the deny-by-default Policy for routes, methods, and write keys.
type AccessPolicy struct {
	allowedStatuses []string
}

func NewAccessPolicy(allowedStatuses []string) *AccessPolicy {
	if len(allowedStatuses) == 0 {
		allowedStatuses = []string{"draft", "publish", "pending"}
	}
	return &AccessPolicy{allowedStatuses: allowedStatuses}
}

type routeRule struct {
	path    *regexp.Regexp
	methods map[string]struct{}
}

func (p *AccessPolicy) rules() []routeRule {
	return []routeRule{
		{path: regexp.MustCompile(`^/wp-json/wp/v2/posts$`), methods: methodSet("GET", "POST", "PATCH")},
		{path: regexp.MustCompile(`^/wp-json/wp/v2/posts/[0-9]+$`), methods: methodSet("GET", "POST", "PATCH")},
		{path: regexp.MustCompile(`^/wp-json/wp/v2/pages$`), methods: methodSet("GET", "POST", "PATCH")},
		{path: regexp.MustCompile(`^/wp-json/wp/v2/pages/[0-9]+$`), methods: methodSet("GET", "POST", "PATCH")},
		{path: regexp.MustCompile(`^/wp-json/wc/v3/products$`), methods: methodSet("GET", "PATCH")},
		{path: regexp.MustCompile(`^/wp-json/wc/v3/products/[0-9]+$`), methods: methodSet("GET", "PATCH")},
		{path: regexp.MustCompile(`^/wp-json/wc/v3/coupons$`), methods: methodSet("GET", "POST")},
		{path: regexp.MustCompile(`^/wp-json/wc/v3/coupons/[0-9]+$`), methods: methodSet("GET", "PATCH", "DELETE")},
		{path: regexp.MustCompile(`^/wp-json/wp/v2/media$`), methods: methodSet("GET", "POST")},
		{path: regexp.MustCompile(`^/wp-json/wp/v2/media/[0-9]+$`), methods: methodSet("GET")},
	}
}

func methodSet(methods ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(methods))
	for _, method := range methods {
		m[method] = struct{}{}
	}
	return m
}

func (p *AccessPolicy) allowedKeys(resource Resource) map[string]struct{} {
	switch resource {
	case ResourcePost:
		return setOf("id", "title", "content", "excerpt", "slug", "status", "featured_media", "categories", "tags")
	case ResourcePage:
		return setOf("id", "title", "content", "excerpt", "slug", "status", "featured_media")
	case ResourceProduct:
		return setOf("id", "name", "slug", "description", "short_description", "images", "catalog_visibility", "categories", "tags")
	case ResourceCoupon:
		return setOf("id", "code", "discount_type", "amount", "date_expires", "date_expires_gmt", "product_ids", "product_categories", "human_confirmed")
	case ResourceMedia:
		return setOf("filename", "content_base64", "alt_text", "mime_type")
	case ResourceGet:
		return setOf("id")
	case ResourceList:
		return map[string]struct{}{}
	default:
		return map[string]struct{}{}
	}
}

func setOf(keys ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		m[k] = struct{}{}
	}
	return m
}

func JSONObjectKeys(raw json.RawMessage) ([]string, error) {
	if len(bytesTrim(raw)) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	return keys, nil
}

func bytesTrim(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

func (p *AccessPolicy) AssertRequest(method, rawURL string) error {
	path := rawURL
	if u, err := url.Parse(rawURL); err == nil && u.Path != "" {
		path = u.Path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	method = strings.ToUpper(method)
	matchedPath := false
	for _, rule := range p.rules() {
		if !rule.path.MatchString(path) {
			continue
		}
		matchedPath = true
		if _, ok := rule.methods[method]; ok {
			return nil
		}
	}
	if matchedPath {
		return NewError(MethodForbidden)
	}
	return NewError(RouteForbidden)
}

func (p *AccessPolicy) AssertWriteKeys(resource Resource, keys []string) error {
	allowed := p.allowedKeys(resource)
	for _, key := range keys {
		if _, ok := allowed[key]; !ok {
			return NewError(FieldForbidden)
		}
		if key == "status" {
			continue
		}
	}
	return nil
}

func (p *AccessPolicy) AssertStatus(status string) error {
	if status == "" {
		return nil
	}
	for _, s := range p.allowedStatuses {
		if status == s {
			return nil
		}
	}
	return NewError(FieldForbidden)
}

func (p *AccessPolicy) AssertCatalogVisibility(v string) error {
	if v == "" {
		return nil
	}
	switch v {
	case "visible", "search", "hidden":
		return nil
	default:
		return NewError(FieldForbidden)
	}
}

func (p *AccessPolicy) AssertDiscountType(v string) error {
	if v == "" {
		return nil
	}
	switch v {
	case "percent", "fixed_cart", "fixed_product":
		return nil
	default:
		return NewError(FieldForbidden)
	}
}

func (p *AccessPolicy) AssertCouponDiscount(discountType, amount string, confirmed bool) error {
	if !strings.EqualFold(discountType, "percent") {
		return nil
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(amount), 64)
	if err != nil {
		return NewError(FieldForbidden)
	}
	if n > 20 && !confirmed {
		return NewError(DiscountConfirmationRequired)
	}
	return nil
}
