package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/auth"
)

func TestTimeAndTimezoneEndpoints(t *testing.T) {
	authSvc, err := auth.NewStatic("admin", "secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	etc := t.TempDir()
	zone := t.TempDir()
	localtime := filepath.Join(t.TempDir(), "localtime")
	if err := os.WriteFile(localtime, []byte("placeholder"), 0o644); err != nil {
		t.Fatal(err)
	}

	mustMkZone(t, zone, "Europe/Warsaw", []byte("warsaw-zone"))
	mustMkZone(t, zone, "Etc/UTC", []byte("utc-zone"))
	mustMkZone(t, zone, "America/New_York", []byte("ny-zone"))

	var setEpoch int64
	prevSet := setSystemClock
	prevBind := rebindLocaltime
	t.Cleanup(func() {
		setSystemClock = prevSet
		rebindLocaltime = prevBind
	})
	setSystemClock = func(epoch int64) error {
		setEpoch = epoch
		return nil
	}
	rebindLocaltime = func(src, dst string) error {
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0o644)
	}

	cfg := Config{
		Auth:          authSvc,
		EtcDir:        etc,
		ZoneinfoDir:   zone,
		LocaltimePath: localtime,
		SecureCookie:  false,
	}
	h := NewRouter(cfg)

	cookie := loginCookie(t, h, "admin", "secret")

	t.Run("get_time", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/time", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
		var body timeResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Epoch <= 0 || body.ISO == "" {
			t.Fatalf("unexpected body: %+v", body)
		}
	})

	t.Run("set_time_epoch", func(t *testing.T) {
		epoch := int64(1722800000)
		payload, _ := json.Marshal(map[string]any{"epoch": epoch})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/time", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
		if setEpoch != epoch {
			t.Fatalf("setEpoch=%d want %d", setEpoch, epoch)
		}
		raw, err := os.ReadFile(filepath.Join(etc, "fake-hwclock"))
		if err != nil {
			t.Fatal(err)
		}
		if string(bytes.TrimSpace(raw)) != "1722800000" {
			t.Fatalf("fake-hwclock=%q", raw)
		}
	})

	t.Run("get_timezone", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(etc, "timezone"), []byte("Europe/Warsaw\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodGet, "/api/v1/timezone", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
		var body timezoneResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Timezone != "Europe/Warsaw" {
			t.Fatalf("timezone=%q", body.Timezone)
		}
		if len(body.Available) < 3 {
			t.Fatalf("available=%v", body.Available)
		}
	})

	t.Run("set_timezone", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]string{"timezone": "America/New_York"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/timezone", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
		raw, err := os.ReadFile(filepath.Join(etc, "timezone"))
		if err != nil {
			t.Fatal(err)
		}
		if string(bytes.TrimSpace(raw)) != "America/New_York" {
			t.Fatalf("timezone file=%q", raw)
		}
		lt, err := os.ReadFile(localtime)
		if err != nil {
			t.Fatal(err)
		}
		if string(lt) != "ny-zone" {
			t.Fatalf("localtime=%q", lt)
		}
	})

	t.Run("set_timezone_unknown", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]string{"timezone": "Mars/Olympus"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/timezone", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func mustMkZone(t *testing.T, root, name string, data []byte) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func loginCookie(t *testing.T, h http.Handler, user, pass string) *http.Cookie {
	t.Helper()
	payload, _ := json.Marshal(map[string]string{"username": user, "password": pass})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("login status %d: %s", rr.Code, rr.Body.String())
	}
	for _, c := range rr.Result().Cookies() {
		if c.Name == auth.SessionCookieName {
			return c
		}
	}
	t.Fatal("no session cookie")
	return nil
}
