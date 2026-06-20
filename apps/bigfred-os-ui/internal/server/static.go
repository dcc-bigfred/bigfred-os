package server

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

func spaHandler(static fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "." || name == "/" {
			name = "index.html"
		}

		data, err := fs.ReadFile(static, name)
		if err != nil {
			data, err = fs.ReadFile(static, "index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			name = "index.html"
		}

		if ct := mime.TypeByExtension(path.Ext(name)); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		_, _ = w.Write(data)
	})
}
