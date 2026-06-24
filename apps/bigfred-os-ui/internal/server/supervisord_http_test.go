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

func TestSupervisordAPI(t *testing.T) {
	dir := t.TempDir()
	conf := filepath.Join(dir, "supervisord.conf")
	if err := os.WriteFile(conf, []byte(`[group:infra]
programs=redis

[program:redis]
command=/bin/bash -c 'valkey-server'
autostart=true
`), 0o644); err != nil {
		t.Fatal(err)
	}

	fakeCtl := filepath.Join(dir, "supervisorctl")
	script := `#!/bin/sh
while [ "$1" = "-c" ]; do shift; shift; done
case "$1" in
status) echo "infra:redis RUNNING pid 7" ;;
start|stop|restart) exit 0 ;;
esac
`
	if err := os.WriteFile(fakeCtl, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	authSvc, err := auth.NewStatic("admin", "secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	h := NewRouter(Config{
		Auth:            authSvc,
		SupervisordConf: conf,
		LogRoots:        []string{t.TempDir()},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"admin","password":"secret"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	cookie := rec.Result().Cookies()[0]

	req = httptest.NewRequest(http.MethodGet, "/api/v1/supervisord/programs", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status: %d body=%s", rec.Code, rec.Body.String())
	}
	var list []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "redis" || list[0].Status != "RUNNING" {
		t.Fatalf("list: %+v", list)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/supervisord/programs/redis/restart", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("restart status: %d body=%s", rec.Code, rec.Body.String())
	}
}
