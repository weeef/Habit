package server

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestStaticFiles(t *testing.T) {
	js := []byte("console.log('hello');\n" + string(make([]byte, 2000)))
	files, err := loadStatic(fstest.MapFS{
		"index.html": {Data: []byte("<!doctype html><title>Habit</title>")},
		"app.js":     {Data: js},
	})
	if err != nil {
		t.Fatal(err)
	}
	get := func(path string, hdr map[string]string) *http.Response {
		req := httptest.NewRequest("GET", path, nil)
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		files.ServeHTTP(w, req)
		return w.Result()
	}

	// Block links and / are the page.
	for _, p := range []string{"/", "/h/gaming"} {
		if res := get(p, nil); res.StatusCode != 200 || res.Header.Get("Content-Type") != "text/html; charset=utf-8" {
			t.Errorf("%s: got %d %q", p, res.StatusCode, res.Header.Get("Content-Type"))
		}
	}
	if res := get("/missing.js", nil); res.StatusCode != 404 {
		t.Errorf("missing file: got %d", res.StatusCode)
	}

	// Gzipped for browsers that ask, and it unzips to the original.
	res := get("/app.js", map[string]string{"Accept-Encoding": "gzip, br"})
	if res.Header.Get("Content-Encoding") != "gzip" {
		t.Fatalf("want gzip, got headers %v", res.Header)
	}
	zr, err := gzip.NewReader(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := io.ReadAll(zr); string(got) != string(js) {
		t.Fatal("gzipped body doesn't match the file")
	}
	if plain := get("/app.js", nil); plain.Header.Get("Content-Encoding") != "" {
		t.Error("shouldn't gzip without Accept-Encoding")
	}

	// A repeat visit with the ETag gets a 304 and no body.
	etag := res.Header.Get("ETag")
	if etag == "" || res.Header.Get("Cache-Control") != "no-cache" {
		t.Fatalf("want an ETag and no-cache, got %v", res.Header)
	}
	if again := get("/app.js", map[string]string{"If-None-Match": etag}); again.StatusCode != http.StatusNotModified {
		t.Errorf("want 304 for a matching ETag, got %d", again.StatusCode)
	}
}
