package wordpress

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
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

func (c *Client) ListPosts(ctx context.Context) ([]PostListItem, error) {
	b, err := c.caller.Do(ctx, http.MethodGet, "/wp-json/wp/v2/posts", nil, "")
	if err != nil {
		return nil, err
	}
	var items []PostListItem
	if err := json.Unmarshal(b, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *Client) GetPost(ctx context.Context, id int) (*Post, error) {
	b, err := c.caller.Do(ctx, http.MethodGet, "/wp-json/wp/v2/posts/"+strconv.Itoa(id), nil, "")
	if err != nil {
		return nil, err
	}
	var post Post
	if err := json.Unmarshal(b, &post); err != nil {
		return nil, err
	}
	return &post, nil
}

func (c *Client) UpsertPost(ctx context.Context, id int, raw json.RawMessage) (*Post, error) {
	body, err := c.prepareWrite(guard.ResourcePost, id, raw)
	if err != nil {
		return nil, err
	}
	method, path := http.MethodPost, "/wp-json/wp/v2/posts"
	if id > 0 {
		method, path = http.MethodPatch, "/wp-json/wp/v2/posts/"+strconv.Itoa(id)
	}
	b, err := c.caller.Do(ctx, method, path, body, "application/json")
	if err != nil {
		return nil, err
	}
	var post Post
	if err := json.Unmarshal(b, &post); err != nil {
		return nil, err
	}
	return &post, nil
}

func (c *Client) ListPages(ctx context.Context) ([]PageListItem, error) {
	b, err := c.caller.Do(ctx, http.MethodGet, "/wp-json/wp/v2/pages", nil, "")
	if err != nil {
		return nil, err
	}
	var items []PageListItem
	if err := json.Unmarshal(b, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *Client) GetPage(ctx context.Context, id int) (*Page, error) {
	b, err := c.caller.Do(ctx, http.MethodGet, "/wp-json/wp/v2/pages/"+strconv.Itoa(id), nil, "")
	if err != nil {
		return nil, err
	}
	var page Page
	if err := json.Unmarshal(b, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

func (c *Client) UpsertPage(ctx context.Context, id int, raw json.RawMessage) (*Page, error) {
	body, err := c.prepareWrite(guard.ResourcePage, id, raw)
	if err != nil {
		return nil, err
	}
	method, path := http.MethodPost, "/wp-json/wp/v2/pages"
	if id > 0 {
		method, path = http.MethodPatch, "/wp-json/wp/v2/pages/"+strconv.Itoa(id)
	}
	b, err := c.caller.Do(ctx, method, path, body, "application/json")
	if err != nil {
		return nil, err
	}
	var page Page
	if err := json.Unmarshal(b, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

func (c *Client) UploadMedia(ctx context.Context, filename string, data []byte, altText, mimeType string) (*Media, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(data); err != nil {
		return nil, err
	}
	if altText != "" {
		_ = w.WriteField("alt_text", altText)
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	ct := w.FormDataContentType()
	if mimeType != "" {
		ct = w.FormDataContentType()
	}
	b, err := c.caller.Do(ctx, http.MethodPost, "/wp-json/wp/v2/media", buf.Bytes(), ct)
	if err != nil {
		return nil, err
	}
	var media Media
	if err := json.Unmarshal(b, &media); err != nil {
		return nil, err
	}
	return &media, nil
}

func (c *Client) GetMedia(ctx context.Context, id int) (*Media, error) {
	b, err := c.caller.Do(ctx, http.MethodGet, "/wp-json/wp/v2/media/"+strconv.Itoa(id), nil, "")
	if err != nil {
		return nil, err
	}
	var media Media
	if err := json.Unmarshal(b, &media); err != nil {
		return nil, err
	}
	return &media, nil
}

func (c *Client) prepareWrite(resource guard.Resource, id int, raw json.RawMessage) ([]byte, error) {
	keys, err := guard.JSONObjectKeys(raw)
	if err != nil {
		return nil, err
	}
	if err := c.policy.AssertWriteKeys(resource, keys); err != nil {
		return nil, err
	}
	var write PostWrite
	if err := json.Unmarshal(raw, &write); err != nil {
		return nil, err
	}
	if write.Status != nil {
		if err := c.policy.AssertStatus(*write.Status); err != nil {
			return nil, err
		}
	} else if id == 0 {
		draft := "draft"
		write.Status = &draft
	}
	body, err := json.Marshal(write)
	if err != nil {
		return nil, err
	}
	if !json.Valid(body) {
		return nil, fmt.Errorf("invalid write body")
	}
	return body, nil
}
