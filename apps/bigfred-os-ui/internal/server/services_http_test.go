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

func TestServicesAPI(t *testing.T) {
	initDir := t.TempDir()
	script := filepath.Join(initDir, "S30-redis")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ncase \"$1\" in start) exit 0;; esac\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	authSvc, err := auth.New("admin", "secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	h := NewRouter(Config{
		Auth:     authSvc,
		InitDir:  initDir,
		LogRoots: []string{t.TempDir()},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"admin","password":"secret"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	cookie := rec.Result().Cookies()[0]

	req = httptest.NewRequest(http.MethodGet, "/api/v1/services", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status: %d", rec.Code)
	}
	var list []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "redis" {
		t.Fatalf("list: %+v", list)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/services/redis/start", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("start status: %d body=%s", rec.Code, rec.Body.String())
	}
}
