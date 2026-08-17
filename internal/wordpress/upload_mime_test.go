package wordpress

import (
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
)

func TestUploadMediaPartUsesMimeType(t *testing.T) {
	var partType, filename string
	client, _ := newWPClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Fatal(err)
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			if part.FormName() == "file" {
				partType = part.Header.Get("Content-Type")
				filename = part.FileName()
				_, _ = io.Copy(io.Discard, part)
			}
		}
		w.Write([]byte(`{"id":44,"mime_type":"image/png"}`))
	}))
	if _, err := client.UploadMedia(context.Background(), "probe.png", []byte("img"), "alt", "image/png"); err != nil {
		t.Fatal(err)
	}
	if partType != "image/png" {
		t.Fatalf("part content-type=%q", partType)
	}
	if filename != "probe.png" {
		t.Fatalf("filename=%q", filename)
	}
}

func TestUploadMediaRejectsPathInFilename(t *testing.T) {
	var filename string
	client, _ := newWPClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Fatal(err)
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			if part.FormName() == "file" {
				filename = part.FileName()
			}
		}
		w.Write([]byte(`{"id":1}`))
	}))
	if _, err := client.UploadMedia(context.Background(), "../../etc/passwd.png", []byte("x"), "", "image/png"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		t.Fatalf("filename leaked path: %q", filename)
	}
}
