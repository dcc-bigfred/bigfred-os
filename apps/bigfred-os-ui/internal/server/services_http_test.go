package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/auth"
	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/microinit"
	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/services"
)

type fakeMicroinit struct{}

func (fakeMicroinit) List() ([]microinit.ServiceStatus, error) {
	pid := int32(1)
	return []microinit.ServiceStatus{{
		Name: "redis", State: "running", PID: &pid, Enabled: true,
	}}, nil
}

func (fakeMicroinit) Control(name, action string) error {
	if name != "redis" {
		return services.ErrNotFound
	}
	switch action {
	case "start", "stop", "restart":
		return nil
	default:
		return services.ErrInvalidAction
	}
}

func TestServicesAPI(t *testing.T) {
	authSvc, err := auth.NewStatic("admin", "secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	h := NewRouter(Config{
		Auth:      authSvc,
		Microinit: fakeMicroinit{},
		LogRoots:  []string{t.TempDir()},
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
		t.Fatalf("list status: %d body=%s", rec.Code, rec.Body.String())
	}
	var list []struct {
		ID      string `json:"id"`
		State   string `json:"state"`
		Running bool   `json:"running"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "redis" || !list[0].Running {
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
