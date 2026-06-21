package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/auth"
	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/redis"
)

func TestRedisAPI(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()
	mr.Set("bigfred:test", "value")

	authSvc, err := auth.New("admin", "secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	h := NewRouter(Config{
		Auth:      authSvc,
		LogRoots:  []string{t.TempDir()},
		RedisAddr: mr.Addr(),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"admin","password":"secret"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	cookie := rec.Result().Cookies()[0]

	req = httptest.NewRequest(http.MethodGet, "/api/v1/redis/keys?pattern=bigfred:*", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status: %d %s", rec.Code, rec.Body.String())
	}
	var keys []redis.KeySummary
	if err := json.NewDecoder(rec.Body).Decode(&keys); err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0].Key != "bigfred:test" {
		t.Fatalf("keys: %+v", keys)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/redis/key?key=bigfred%3Atest", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status: %d %s", rec.Code, rec.Body.String())
	}
	var detail redis.KeyDetail
	if err := json.NewDecoder(rec.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if detail.Value != "value" {
		t.Fatalf("detail: %+v", detail)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/redis/key?key=bigfred%3Atest", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status: %d %s", rec.Code, rec.Body.String())
	}
}
