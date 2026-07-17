package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/auth"
	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/update"
)

func TestUpdateEndpoint(t *testing.T) {
	authSvc, err := auth.NewStatic("admin", "secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	payload := []byte("binary-bytes")
	updater := update.New(update.Config{
		InstallDir: dir,
		Arch:       "arm64",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/releases/tags/") || strings.Contains(req.URL.Path, "/releases/latest") {
				body := `{"tag_name":"v2","assets":[{"name":"loco-server-linux-arm64","browser_download_url":"http://example.test/bin","size":12}]}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(string(payload))),
				Header:     make(http.Header),
			}, nil
		})},
	})

	h := NewRouter(Config{Auth: authSvc, Updater: updater})

	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"admin","password":"secret"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, login)
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d", rec.Code)
	}
	cookie := rec.Result().Cookies()[0]

	req := httptest.NewRequest(http.MethodPost, "/api/v1/update/bigfred", strings.NewReader(`{"tag":"v2"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status: %d %s", rec.Code, rec.Body.String())
	}
	var res update.Result
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatal(err)
	}
	if res.Tag != "v2" || res.Restart != "bigfred" {
		t.Fatalf("res=%+v", res)
	}
	got, err := os.ReadFile(filepath.Join(dir, "bigfred"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("payload=%q", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
