package guard

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

type stubAuth struct {
	user string
	pass string
}

func (s stubAuth) Apply(req *http.Request) {
	req.SetBasicAuth(s.user, s.pass)
}

func TestStoreCallerGETRetriesOnceOn500(t *testing.T) {
	var hits atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte(`{"id":1}`))
	}))
	defer ts.Close()
	c := NewStoreCaller(ts.URL, stubAuth{user: "u", pass: "test-key-123"}, NewAccessPolicy(nil))
	body, err := c.Do(context.Background(), http.MethodGet, "/wp-json/wp/v2/posts/1", nil, "")
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if hits.Load() != 2 {
		t.Fatalf("hits=%d want 2", hits.Load())
	}
	if !strings.Contains(string(body), `"id":1`) {
		t.Fatalf("body=%s", body)
	}
}

func TestStoreCallerPOSTDoesNotRetry(t *testing.T) {
	var hits atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()
	c := NewStoreCaller(ts.URL, stubAuth{user: "u", pass: "test-key-123"}, NewAccessPolicy(nil))
	_, err := c.Do(context.Background(), http.MethodPost, "/wp-json/wp/v2/posts", []byte(`{"title":"x"}`), "application/json")
	if err == nil {
		t.Fatal("expected error")
	}
	if hits.Load() != 1 {
		t.Fatalf("hits=%d want 1", hits.Load())
	}
}

func TestStoreCallerRefusesOffHostRedirect(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://evil.example/steal", http.StatusFound)
	}))
	defer ts.Close()
	c := NewStoreCaller(ts.URL, stubAuth{user: "u", pass: "test-key-123"}, NewAccessPolicy(nil))
	_, err := c.Do(context.Background(), http.MethodGet, "/wp-json/wp/v2/posts/1", nil, "")
	if err == nil {
		t.Fatal("expected redirect error")
	}
}

func TestStoreCallerSendsBasicAuthAndDoesNotLogSecret(t *testing.T) {
	secret := "test-key-123"
	var gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Query().Get("consumer_key") != "" {
			t.Errorf("consumer_key in query")
		}
		w.Write([]byte(`{"id":1}`))
	}))
	defer ts.Close()
	var logBuf strings.Builder
	c := NewStoreCaller(ts.URL, stubAuth{user: "mcp-content", pass: secret}, NewAccessPolicy(nil))
	c.Logger = slog.New(slog.NewTextHandler(&logBuf, nil))
	c.StoreID = "minha-loja"
	_, err := c.Do(context.Background(), http.MethodGet, "/wp-json/wp/v2/posts/1", nil, "")
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if gotAuth == "" {
		t.Fatal("missing Authorization")
	}
	if strings.Contains(logBuf.String(), secret) || strings.Contains(logBuf.String(), gotAuth) {
		t.Fatalf("secret or Authorization in logs: %s", logBuf.String())
	}
}

func TestStoreCallerNilAuthIsAuthMissing(t *testing.T) {
	c := NewStoreCaller("https://loja.example.com", nil, NewAccessPolicy(nil))
	_, err := c.Do(context.Background(), http.MethodGet, "/wp-json/wp/v2/posts", nil, "")
	if !IsCode(err, AuthMissing) {
		t.Fatalf("got %v, want AUTH_MISSING", err)
	}
}

func TestStoreCallerUsersNeverSent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("HTTP was sent: %s %s", r.Method, r.URL.Path)
		io.WriteString(w, "{}")
	}))
	defer ts.Close()
	c := NewStoreCaller(ts.URL, stubAuth{user: "u", pass: "test-key-123"}, NewAccessPolicy(nil))
	_, err := c.Do(context.Background(), http.MethodGet, "/wp-json/wp/v2/users", nil, "")
	if !IsCode(err, RouteForbidden) {
		t.Fatalf("got %v, want ROUTE_FORBIDDEN", err)
	}
}
