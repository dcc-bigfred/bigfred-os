package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/auth"
	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/etcdir"
)

func TestEtcFilesAPI(t *testing.T) {
	etcRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(etcRoot, "fanctl.conf"), []byte("old\n"), 0o640); err != nil {
		t.Fatal(err)
	}

	authSvc, err := auth.New("admin", "secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	h := NewRouter(Config{
		Auth:     authSvc,
		EtcDir:   etcRoot,
		LogRoots: []string{t.TempDir()},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"admin","password":"secret"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	cookie := rec.Result().Cookies()[0]

	req = httptest.NewRequest(http.MethodGet, "/api/v1/etc/files", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status: %d %s", rec.Code, rec.Body.String())
	}
	var list []etcdir.Entry
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Path != "fanctl.conf" {
		t.Fatalf("list: %+v", list)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/etc/file?path=fanctl.conf", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("read status: %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPut, "/api/v1/etc/file?path=fanctl.conf", strings.NewReader(`{"content":"new\n"}`))
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("write status: %d %s", rec.Code, rec.Body.String())
	}
	var body etcdir.FileContent
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Content != "new\n" {
		t.Fatalf("content: %q", body.Content)
	}
}
