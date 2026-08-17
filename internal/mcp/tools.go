package mcpserver

import (
	"context"
	"encoding/base64"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"woocommerce-store-mcp/internal/guard"
)

func inputSchema[T any]() *jsonschema.Schema {
	s, err := jsonschema.For[T](nil)
	if err != nil {
		panic(err)
	}
	s.AdditionalProperties = &jsonschema.Schema{}
	return s
}

func (r *Runtime) registerTools() {
	mcp.AddTool(r.server, &mcp.Tool{Name: "list_posts", Description: "Lista posts (id, title, slug, status, link). Não filtrar por autor/email.", InputSchema: inputSchema[emptyInput]()}, r.listPosts)
	mcp.AddTool(r.server, &mcp.Tool{Name: "get_post", Description: "Lê um post editorial. Não devolve password nem author email.", InputSchema: inputSchema[idInput]()}, r.getPost)
	mcp.AddTool(r.server, &mcp.Tool{Name: "upsert_post", Description: "Cria ou actualiza post (title, content, excerpt, slug, status draft|publish|pending, featured_media, categories, tags). Default create: draft. Não faz DELETE nem author/meta livre. Não publicar conteúdo gerado sem confirmação humana se status for passado.", InputSchema: inputSchema[upsertPostInput]()}, r.upsertPost)
	mcp.AddTool(r.server, &mcp.Tool{Name: "list_pages", Description: "Lista páginas (id, title, slug, status, link).", InputSchema: inputSchema[emptyInput]()}, r.listPages)
	mcp.AddTool(r.server, &mcp.Tool{Name: "get_page", Description: "Lê uma página editorial.", InputSchema: inputSchema[idInput]()}, r.getPage)
	mcp.AddTool(r.server, &mcp.Tool{Name: "upsert_page", Description: "Cria ou actualiza página (title, content, excerpt, slug, status, featured_media). Default create: draft. Não faz DELETE.", InputSchema: inputSchema[upsertPageInput]()}, r.upsertPage)
	mcp.AddTool(r.server, &mcp.Tool{Name: "list_products", Description: "Lista todos os produtos de todas as páginas (id, name, slug, status, permalink). Não devolve preço/stock.", InputSchema: inputSchema[emptyInput]()}, r.listProducts)
	mcp.AddTool(r.server, &mcp.Tool{Name: "get_product_content", Description: "Lê nome, descrições, imagens e slug de produto. Não devolve preço, stock, sku nem meta_data.", InputSchema: inputSchema[idInput]()}, r.getProductContent)
	mcp.AddTool(r.server, &mcp.Tool{Name: "update_product_content", Description: "PATCH conteúdo editorial de produto existente (name, slug, description, short_description, images, catalog_visibility, categories, tags). Não cria nem apaga produto. Não altera preço, stock nem variações. Recusar preço/stock de produto.", InputSchema: inputSchema[updateProductInput]()}, r.updateProductContent)
	mcp.AddTool(r.server, &mcp.Tool{Name: "upload_media", Description: "Upload de imagem para a media library; devolve id. Não liga a settings/logo do site.", InputSchema: inputSchema[uploadMediaInput]()}, r.uploadMedia)
	mcp.AddTool(r.server, &mcp.Tool{Name: "list_coupons", Description: "Lista cupons (id, code, type, amount, validade, âmbito). Não devolve encomendas/clientes.", InputSchema: inputSchema[emptyInput]()}, r.listCoupons)
	mcp.AddTool(r.server, &mcp.Tool{Name: "create_coupon", Description: "Cria cupom (code, discount_type percent|fixed_cart|fixed_product, amount, date_expires, product_ids, product_categories). Percentagem > 20 exige AskQuestion e human_confirmed=true. Não usa sale_price em produto.", InputSchema: couponInputSchema()}, r.createCoupon)
	mcp.AddTool(r.server, &mcp.Tool{Name: "update_coupon", Description: "PATCH cupom. Percentagem > 20 exige AskQuestion e human_confirmed=true.", InputSchema: couponInputSchema()}, r.updateCoupon)
	mcp.AddTool(r.server, &mcp.Tool{Name: "delete_coupon", Description: "DELETE /wc/v3/coupons/{id}. Não apaga posts, páginas, produtos nem media.", InputSchema: inputSchema[idInput]()}, r.deleteCoupon)
}

func (r *Runtime) listPosts(ctx context.Context, req *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, any, error) {
	if err := r.assertKeys(guard.ResourceList, req.Params.Arguments); err != nil {
		return nil, nil, err
	}
	r.logTool("list_posts", 0)
	return packList(r.wp.ListPosts(ctx))
}

func (r *Runtime) getPost(ctx context.Context, req *mcp.CallToolRequest, in idInput) (*mcp.CallToolResult, any, error) {
	if err := r.assertKeys(guard.ResourceGet, req.Params.Arguments); err != nil {
		return nil, nil, err
	}
	r.logTool("get_post", in.ID)
	return pack(r.wp.GetPost(ctx, in.ID))
}

func (r *Runtime) upsertPost(ctx context.Context, req *mcp.CallToolRequest, in upsertPostInput) (*mcp.CallToolResult, any, error) {
	if err := r.assertKeys(guard.ResourcePost, req.Params.Arguments); err != nil {
		return nil, nil, err
	}
	r.logTool("upsert_post", in.ID)
	return pack(r.wp.UpsertPost(ctx, in.ID, req.Params.Arguments))
}

func (r *Runtime) listPages(ctx context.Context, req *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, any, error) {
	if err := r.assertKeys(guard.ResourceList, req.Params.Arguments); err != nil {
		return nil, nil, err
	}
	r.logTool("list_pages", 0)
	return packList(r.wp.ListPages(ctx))
}

func (r *Runtime) getPage(ctx context.Context, req *mcp.CallToolRequest, in idInput) (*mcp.CallToolResult, any, error) {
	if err := r.assertKeys(guard.ResourceGet, req.Params.Arguments); err != nil {
		return nil, nil, err
	}
	r.logTool("get_page", in.ID)
	return pack(r.wp.GetPage(ctx, in.ID))
}

func (r *Runtime) upsertPage(ctx context.Context, req *mcp.CallToolRequest, in upsertPageInput) (*mcp.CallToolResult, any, error) {
	if err := r.assertKeys(guard.ResourcePage, req.Params.Arguments); err != nil {
		return nil, nil, err
	}
	r.logTool("upsert_page", in.ID)
	return pack(r.wp.UpsertPage(ctx, in.ID, req.Params.Arguments))
}

func (r *Runtime) listProducts(ctx context.Context, req *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, any, error) {
	if err := r.assertKeys(guard.ResourceList, req.Params.Arguments); err != nil {
		return nil, nil, err
	}
	r.logTool("list_products", 0)
	return packList(r.wc.ListProducts(ctx))
}

func (r *Runtime) getProductContent(ctx context.Context, req *mcp.CallToolRequest, in idInput) (*mcp.CallToolResult, any, error) {
	if err := r.assertKeys(guard.ResourceGet, req.Params.Arguments); err != nil {
		return nil, nil, err
	}
	r.logTool("get_product_content", in.ID)
	return pack(r.wc.GetProductContent(ctx, in.ID))
}

func (r *Runtime) updateProductContent(ctx context.Context, req *mcp.CallToolRequest, in updateProductInput) (*mcp.CallToolResult, any, error) {
	if err := r.assertKeys(guard.ResourceProduct, req.Params.Arguments); err != nil {
		return nil, nil, err
	}
	r.logTool("update_product_content", in.ID)
	return pack(r.wc.UpdateProductContent(ctx, in.ID, req.Params.Arguments))
}

func (r *Runtime) uploadMedia(ctx context.Context, req *mcp.CallToolRequest, in uploadMediaInput) (*mcp.CallToolResult, any, error) {
	if err := r.assertKeys(guard.ResourceMedia, req.Params.Arguments); err != nil {
		return nil, nil, err
	}
	data, err := base64.StdEncoding.DecodeString(in.ContentBase64)
	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(in.ContentBase64)
		if err != nil {
			return nil, nil, err
		}
	}
	r.logTool("upload_media", 0)
	return pack(r.wp.UploadMedia(ctx, in.Filename, data, in.AltText, in.MimeType))
}

func (r *Runtime) listCoupons(ctx context.Context, req *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, any, error) {
	if err := r.assertKeys(guard.ResourceList, req.Params.Arguments); err != nil {
		return nil, nil, err
	}
	r.logTool("list_coupons", 0)
	return packList(r.wc.ListCoupons(ctx))
}

func (r *Runtime) createCoupon(ctx context.Context, req *mcp.CallToolRequest, in couponInput) (*mcp.CallToolResult, any, error) {
	if err := r.assertKeys(guard.ResourceCoupon, req.Params.Arguments); err != nil {
		return nil, nil, err
	}
	r.logTool("create_coupon", 0)
	return pack(r.wc.CreateCoupon(ctx, req.Params.Arguments, in.HumanConfirmed))
}

func (r *Runtime) updateCoupon(ctx context.Context, req *mcp.CallToolRequest, in couponInput) (*mcp.CallToolResult, any, error) {
	if err := r.assertKeys(guard.ResourceCoupon, req.Params.Arguments); err != nil {
		return nil, nil, err
	}
	r.logTool("update_coupon", in.ID)
	return pack(r.wc.UpdateCoupon(ctx, in.ID, req.Params.Arguments, in.HumanConfirmed))
}

func (r *Runtime) deleteCoupon(ctx context.Context, req *mcp.CallToolRequest, in idInput) (*mcp.CallToolResult, any, error) {
	if err := r.assertKeys(guard.ResourceGet, req.Params.Arguments); err != nil {
		return nil, nil, err
	}
	r.logTool("delete_coupon", in.ID)
	return pack(r.wc.DeleteCoupon(ctx, in.ID))
}

func pack[T any](v T, err error) (*mcp.CallToolResult, any, error) {
	return nil, v, err
}

func packList[T any](v []T, err error) (*mcp.CallToolResult, any, error) {
	if err != nil {
		return nil, nil, err
	}
	if v == nil {
		v = []T{}
	}
	return nil, map[string]any{"items": v}, nil
}

func couponInputSchema() *jsonschema.Schema {
	s := inputSchema[couponInput]()
	if p := s.Properties["amount"]; p != nil {
		p.Type = ""
		p.Types = []string{"string", "number"}
	}
	return s
}
