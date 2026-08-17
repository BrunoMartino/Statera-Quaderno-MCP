package wordpress

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
)

func TestListPagesWalksAllPages(t *testing.T) {
	var pages []int
	client, _ := newWPClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/wp-json/wp/v2/pages" {
			t.Fatalf("got %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("per_page") != "100" {
			t.Fatalf("per_page=%s", r.URL.Query().Get("per_page"))
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pages = append(pages, page)
		n := 100
		base := 1
		if page == 2 {
			n = 3
			base = 101
		}
		items := make([]PageListItem, n)
		for i := 0; i < n; i++ {
			items[i] = PageListItem{ID: base + i, Slug: "p", Status: "publish"}
		}
		json.NewEncoder(w).Encode(items)
	}))
	got, err := client.ListPages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 103 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].ID != 1 || got[102].ID != 103 {
		t.Fatalf("ids %d ... %d", got[0].ID, got[102].ID)
	}
	if len(pages) != 2 || pages[0] != 1 || pages[1] != 2 {
		t.Fatalf("pages=%v", pages)
	}
}

func TestListPostsWalksAllPagesAndEditorialStatuses(t *testing.T) {
	var pages []int
	client, _ := newWPClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wp-json/wp/v2/posts" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("per_page") != "100" {
			t.Fatalf("per_page=%s", r.URL.Query().Get("per_page"))
		}
		if r.URL.Query().Get("status") != "draft,publish,pending" {
			t.Fatalf("status=%s", r.URL.Query().Get("status"))
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pages = append(pages, page)
		n := 100
		base := 1
		if page == 2 {
			n = 2
			base = 101
		}
		items := make([]PostListItem, n)
		for i := 0; i < n; i++ {
			items[i] = PostListItem{ID: base + i, Status: "draft"}
		}
		json.NewEncoder(w).Encode(items)
	}))
	got, err := client.ListPosts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 102 {
		t.Fatalf("len=%d", len(got))
	}
	if len(pages) != 2 {
		t.Fatalf("pages=%v", pages)
	}
}

func TestGetPostRequestsContextEdit(t *testing.T) {
	client, _ := newWPClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wp-json/wp/v2/posts/2" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("context") != "edit" {
			t.Fatalf("context=%s", r.URL.Query().Get("context"))
		}
		w.Write([]byte(`{"id":2,"title":{"raw":"T","rendered":"T"},"content":{"raw":"<p>x</p>","rendered":"<p>x</p>"},"status":"draft"}`))
	}))
	post, err := client.GetPost(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if post.Content.Raw != "<p>x</p>" {
		t.Fatalf("raw=%q", post.Content.Raw)
	}
}

func TestGetPageRequestsContextEdit(t *testing.T) {
	client, _ := newWPClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wp-json/wp/v2/pages/282" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("context") != "edit" {
			t.Fatalf("context=%s", r.URL.Query().Get("context"))
		}
		w.Write([]byte(`{"id":282,"title":{"raw":"Sobre","rendered":"Sobre"},"content":{"raw":"<p>html</p>","rendered":"<p>html</p>"},"status":"publish"}`))
	}))
	page, err := client.GetPage(context.Background(), 282)
	if err != nil {
		t.Fatal(err)
	}
	if page.Content.Raw != "<p>html</p>" {
		t.Fatalf("raw=%q", page.Content.Raw)
	}
}
