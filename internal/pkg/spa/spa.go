// Package spa serves a single-page application embedded in an fs.FS.
package spa

import (
	"bytes"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

const (
	indexFile    = "index.html"
	immutableAge = "public, max-age=31536000, immutable"
	noCache      = "no-cache"
)

// Handler serves fsys under prefix (for example "/admin"). Unknown paths fall
// back to index.html so the client-side router can resolve them, except a
// missing file under assets/, which returns 404. Files under assets/ get a
// long immutable cache; index.html is never cached.
func Handler(fsys fs.FS, prefix string) http.Handler {
	prefix = "/" + strings.Trim(prefix, "/")
	if prefix == "/" {
		prefix = ""
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		upath := r.URL.Path
		if prefix != "" {
			if upath == prefix {
				target := prefix + "/"
				if r.URL.RawQuery != "" {
					target += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, target, http.StatusMovedPermanently) //nolint:gosec // G710: target is a same-origin path built from the configured prefix
				return
			}
			if !strings.HasPrefix(upath, prefix+"/") {
				http.NotFound(w, r)
				return
			}
			upath = strings.TrimPrefix(upath, prefix)
		}

		clean := path.Clean("/" + upath)
		if clean == "/" {
			serveIndex(w, r, fsys)
			return
		}
		name := strings.TrimPrefix(clean, "/")
		info, err := fs.Stat(fsys, name)
		if err != nil || info.IsDir() {
			if strings.HasPrefix(clean, "/assets/") {
				http.NotFound(w, r)
				return
			}
			serveIndex(w, r, fsys)
			return
		}
		serveFile(w, r, fsys, name, info)
	})
}

// serveIndex writes index.html with a no-cache policy.
func serveIndex(w http.ResponseWriter, r *http.Request, fsys fs.FS) {
	info, err := fs.Stat(fsys, indexFile)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	serveFile(w, r, fsys, indexFile, info)
}

// serveFile copies one file, honouring Range requests through ServeContent.
func serveFile(w http.ResponseWriter, r *http.Request, fsys fs.FS, name string, info fs.FileInfo) {
	f, err := fsys.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = f.Close() }()

	w.Header().Set("Cache-Control", cachePolicy(name))
	if ct := mime.TypeByExtension(path.Ext(name)); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	var content io.ReadSeeker
	if rs, ok := f.(io.ReadSeeker); ok {
		content = rs
	} else {
		data, err := io.ReadAll(f)
		if err != nil {
			http.Error(w, "cannot read asset", http.StatusInternalServerError)
			return
		}
		content = bytes.NewReader(data)
	}
	http.ServeContent(w, r, name, info.ModTime(), content)
}

// cachePolicy returns the Cache-Control value for a served file.
func cachePolicy(name string) string {
	if strings.HasPrefix(name, "assets/") {
		return immutableAge
	}
	return noCache
}
