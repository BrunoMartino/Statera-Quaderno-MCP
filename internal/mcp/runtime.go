package mcpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"woocommerce-store-mcp/internal/config"
	"woocommerce-store-mcp/internal/guard"
	"woocommerce-store-mcp/internal/woocommerce"
	"woocommerce-store-mcp/internal/wordpress"
)

type Runtime struct {
	cfg    *config.Config
	policy *guard.AccessPolicy
	wp     *wordpress.Client
	wc     *woocommerce.Client
	server *mcp.Server
	logger *slog.Logger
}

func New(cfg *config.Config, caller *guard.StoreCaller, logger *slog.Logger) *Runtime {
	policy := guard.NewAccessPolicy(nil)
	if cfg != nil {
		policy = guard.NewAccessPolicy(cfg.AllowedStatuses)
	}
	if logger == nil {
		logger = slog.Default()
	}
	r := &Runtime{
		cfg:    cfg,
		policy: policy,
		wp:     wordpress.NewClient(caller, policy),
		wc:     woocommerce.NewClient(caller, policy),
		logger: logger,
	}
	r.server = mcp.NewServer(&mcp.Implementation{
		Name:    "woocommerce-store-mcp",
		Version: "1.0.0",
	}, &mcp.ServerOptions{Instructions: instructions, Logger: logger})
	r.registerTools()
	return r
}

func (r *Runtime) Server() *mcp.Server {
	return r.server
}

func (r *Runtime) HTTPHandler() http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return r.server
	}, nil)
}

func (r *Runtime) logTool(name string, id int) {
	storeID, env := "", ""
	if r.cfg != nil {
		storeID, env = r.cfg.StoreID, r.cfg.Environment
	}
	r.logger.Info("tool", "name", name, "id", id, "store", storeID, "env", env)
}

func (r *Runtime) assertKeys(resource guard.Resource, raw json.RawMessage) error {
	keys, err := guard.JSONObjectKeys(raw)
	if err != nil {
		return err
	}
	return r.policy.AssertWriteKeys(resource, keys)
}

type idInput struct {
	ID int `json:"id"`
}

type upsertPostInput struct {
	ID            int    `json:"id,omitempty"`
	Title         string `json:"title,omitempty"`
	Content       string `json:"content,omitempty"`
	Excerpt       string `json:"excerpt,omitempty"`
	Slug          string `json:"slug,omitempty"`
	Status        string `json:"status,omitempty"`
	FeaturedMedia int    `json:"featured_media,omitempty"`
	Categories    []int  `json:"categories,omitempty"`
	Tags          []int  `json:"tags,omitempty"`
}

type upsertPageInput struct {
	ID            int    `json:"id,omitempty"`
	Title         string `json:"title,omitempty"`
	Content       string `json:"content,omitempty"`
	Excerpt       string `json:"excerpt,omitempty"`
	Slug          string `json:"slug,omitempty"`
	Status        string `json:"status,omitempty"`
	FeaturedMedia int    `json:"featured_media,omitempty"`
}

type updateProductInput struct {
	ID                int                        `json:"id"`
	Name              string                     `json:"name,omitempty"`
	Slug              string                     `json:"slug,omitempty"`
	Description       string                     `json:"description,omitempty"`
	ShortDescription  string                     `json:"short_description,omitempty"`
	Images            []woocommerce.ProductImage `json:"images,omitempty"`
	CatalogVisibility string                     `json:"catalog_visibility,omitempty"`
	Categories        []woocommerce.ProductTerm  `json:"categories,omitempty"`
	Tags              []woocommerce.ProductTerm  `json:"tags,omitempty"`
}

type uploadMediaInput struct {
	Filename      string `json:"filename"`
	ContentBase64 string `json:"content_base64"`
	AltText       string `json:"alt_text,omitempty"`
	MimeType      string `json:"mime_type,omitempty"`
}

type couponInput struct {
	ID                int    `json:"id,omitempty"`
	Code              string `json:"code,omitempty"`
	DiscountType      string `json:"discount_type,omitempty"`
	Amount            string `json:"amount,omitempty"`
	DateExpires       string `json:"date_expires,omitempty"`
	DateExpiresGMT    string `json:"date_expires_gmt,omitempty"`
	ProductIDs        []int  `json:"product_ids,omitempty"`
	ProductCategories []int  `json:"product_categories,omitempty"`
	HumanConfirmed    bool   `json:"human_confirmed,omitempty"`
}

type emptyInput struct{}
