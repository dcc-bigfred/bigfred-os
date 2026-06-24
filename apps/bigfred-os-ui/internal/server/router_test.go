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
)

func TestLoginAndLogsAPI(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "redis")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "redis.log"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	authSvc, err := auth.NewStatic("admin", "secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	h := NewRouter(Config{
		Auth:     authSvc,
		LogRoots: []string{root},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"admin","password":"secret"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status: %d %s", rec.Code, rec.Body.String())
	}
	cookie := rec.Result().Cookies()[0]

	req = httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("logs status: %d", rec.Code)
	}
	var entries []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "redis.log" {
		t.Fatalf("entries: %+v", entries)
	}
}
