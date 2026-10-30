package server

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

// staticFile is one file of the web client, prepared once at startup: its
// bytes, a gzipped copy when that's smaller, and an ETag of its contents.
type staticFile struct {
	body, gz []byte
	ctype    string
	etag     string
}

// staticFiles serves the embedded client. Text files are sent gzipped to
// browsers that accept it (app.js shrinks from ~200 KB to ~60 KB), and every
// file carries an ETag with "no-cache", so browsers keep their copy and only
// download it again after a deploy changes it.
type staticFiles map[string]*staticFile

func loadStatic(fsys fs.FS) (staticFiles, error) {
	files := staticFiles{}
	err := fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		body, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(body)
		f := &staticFile{body: body, etag: `"` + hex.EncodeToString(sum[:8]) + `"`}
		f.ctype = mime.TypeByExtension(path.Ext(name))
		if f.ctype == "" {
			f.ctype = http.DetectContentType(body)
		}
		if compressible(f.ctype) {
			var buf bytes.Buffer
			zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
			zw.Write(body)
			zw.Close()
			if buf.Len() < len(body) {
				f.gz = buf.Bytes()
			}
		}
		files["/"+name] = f
		return nil
	})
	return files, err
}

func compressible(ctype string) bool {
	return strings.HasPrefix(ctype, "text/") || strings.Contains(ctype, "javascript") ||
		strings.Contains(ctype, "json") || strings.Contains(ctype, "svg")
}

// serve writes the file at name (a path like "/app.js"), or a 404.
func (files staticFiles) serve(w http.ResponseWriter, r *http.Request, name string) {
	f := files[name]
	if f == nil {
		http.NotFound(w, r)
		return
	}
	h := w.Header()
	h.Set("Content-Type", f.ctype)
	h.Set("Cache-Control", "no-cache") // keep it, but check it's current
	h.Set("ETag", f.etag)
	h.Set("X-Content-Type-Options", "nosniff")
	if f.gz != nil {
		h.Set("Vary", "Accept-Encoding")
	}
	if match := r.Header.Get("If-None-Match"); match != "" && strings.Contains(match, f.etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	body := f.body
	if f.gz != nil && acceptsGzip(r) {
		h.Set("Content-Encoding", "gzip")
		body = f.gz
	}
	if r.Method == http.MethodHead {
		return
	}
	w.Write(body)
}

func acceptsGzip(r *http.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		enc, q, _ := strings.Cut(strings.TrimSpace(part), ";")
		if strings.TrimSpace(enc) == "gzip" && strings.TrimSpace(q) != "q=0" {
			return true
		}
	}
	return false
}

// ServeHTTP serves the client: "/" is the page itself, block links like
// /h/gaming open the same page (it then joins that block), and anything else
// is a file.
func (files staticFiles) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := r.URL.Path
	if name == "/" || strings.HasPrefix(name, "/h/") {
		name = "/index.html"
	}
	files.serve(w, r, name)
}
