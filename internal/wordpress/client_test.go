package wordpress

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"woocommerce-store-mcp/internal/guard"
)

type recordingAuth struct{}

func (recordingAuth) Apply(req *http.Request) {
	req.SetBasicAuth("mcp-content", "test-key-123")
}

func newWPClient(t *testing.T, h http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	policy := guard.NewAccessPolicy(nil)
	caller := guard.NewStoreCaller(ts.URL, recordingAuth{}, policy)
	return NewClient(caller, policy), ts
}

func TestUpsertPostTitleContentSendsAllowlistAndDraft(t *testing.T) {
	var gotBody []byte
	var gotMethod, gotPath string
	client, _ := newWPClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		w.Write([]byte(`{"id":11,"type":"post","title":{"rendered":"Hi"},"content":{"rendered":"Body"},"status":"draft","slug":"hi","link":"https://loja.example.com/hi","modified":"2026-08-17T12:00:00"}`))
	}))
	post, err := client.UpsertPost(context.Background(), 0, json.RawMessage(`{"title":"Hi","content":"Body"}`))
	if err != nil {
		t.Fatalf("UpsertPost: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/wp-json/wp/v2/posts" {
		t.Fatalf("got %s %s", gotMethod, gotPath)
	}
	if !strings.Contains(string(gotBody), `"status":"draft"`) {
		t.Fatalf("expected default draft, body=%s", gotBody)
	}
	if strings.Contains(string(gotBody), "password") || strings.Contains(string(gotBody), "author") {
		t.Fatalf("forbidden keys in body: %s", gotBody)
	}
	if post.ID != 11 || post.Status != "draft" {
		t.Fatalf("post=%+v", post)
	}
}

func TestUpsertPostPasswordIsFieldForbiddenNoHTTP(t *testing.T) {
	var hits int
	client, _ := newWPClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	_, err := client.UpsertPost(context.Background(), 1, json.RawMessage(`{"title":"x","password":"secret"}`))
	if !guard.IsCode(err, guard.FieldForbidden) {
		t.Fatalf("got %v, want FIELD_FORBIDDEN", err)
	}
	if hits != 0 {
		t.Fatalf("HTTP hits=%d", hits)
	}
}

func TestGetPostOmitsAuthorAndEmail(t *testing.T) {
	client, _ := newWPClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":2,"type":"post","link":"https://loja.example.com/p","title":{"rendered":"T"},"content":{"rendered":"C"},"excerpt":{"rendered":""},"slug":"p","status":"publish","featured_media":0,"modified":"2026-01-01T00:00:00","author":99,"email":"admin@example.com","yoast_head":"<script>"}`))
	}))
	post, err := client.GetPost(context.Background(), 2)
	if err != nil {
		t.Fatalf("GetPost: %v", err)
	}
	encoded, _ := json.Marshal(post)
	if strings.Contains(string(encoded), "admin@example.com") || strings.Contains(string(encoded), `"author"`) {
		t.Fatalf("leaked author/email: %s", encoded)
	}
}

func TestGetPostDoesNotCallUsers(t *testing.T) {
	client, _ := newWPClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "users") {
			t.Errorf("users path hit")
		}
		w.Write([]byte(`{"id":2,"title":{"rendered":"T"},"status":"draft"}`))
	}))
	if _, err := client.GetPost(context.Background(), 2); err != nil {
		t.Fatal(err)
	}
}

func TestUploadMediaReturnsID(t *testing.T) {
	var gotPath, gotMethod string
	client, _ := newWPClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("content-type %s", r.Header.Get("Content-Type"))
		}
		w.Write([]byte(`{"id":44,"source_url":"https://loja.example.com/wp-content/uploads/a.jpg","mime_type":"image/jpeg"}`))
	}))
	media, err := client.UploadMedia(context.Background(), "a.jpg", []byte("fakeimg"), "alt", "image/jpeg")
	if err != nil {
		t.Fatalf("UploadMedia: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/wp-json/wp/v2/media" {
		t.Fatalf("got %s %s", gotMethod, gotPath)
	}
	if media.ID != 44 {
		t.Fatalf("id=%d", media.ID)
	}
}

func TestGetMediaByID(t *testing.T) {
	client, _ := newWPClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wp-json/wp/v2/media/44" {
			t.Errorf("path %s", r.URL.Path)
		}
		w.Write([]byte(`{"id":44}`))
	}))
	media, err := client.GetMedia(context.Background(), 44)
	if err != nil {
		t.Fatal(err)
	}
	if media.ID != 44 {
		t.Fatalf("id=%d", media.ID)
	}
}
