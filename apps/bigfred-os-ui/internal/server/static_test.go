package server

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestSPAHandlerNoRedirectLoop(t *testing.T) {
	static := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>ok</html>")},
		"assets/app.js": &fstest.MapFile{
			Data: []byte("console.log('ok')"),
		},
	}

	h := spaHandler(static)
	for _, path := range []string{"/", "/logs", "/assets/app.js"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code == http.StatusMovedPermanently || rec.Code == http.StatusFound {
			t.Fatalf("%s: unexpected redirect %d Location=%s", path, rec.Code, rec.Header().Get("Location"))
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d", path, rec.Code)
		}
	}
}

func TestSPAHandlerHead(t *testing.T) {
	static := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>ok</html>")},
	}
	h := spaHandler(static)
	req := httptest.NewRequest(http.MethodHead, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty body for HEAD")
	}
}

// Ensure compile-time interface satisfaction for embed use.
var _ fs.FS = fstest.MapFS{}
