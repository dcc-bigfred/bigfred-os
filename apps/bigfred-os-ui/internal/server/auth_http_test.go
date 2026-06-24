package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/auth"
)

func TestChangePasswordAPI(t *testing.T) {
	authSvc, err := auth.NewStatic("root", "oldpass", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	h := NewRouter(Config{Auth: authSvc})

	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"username":"root","password":"oldpass"}`))
	loginRec := httptest.NewRecorder()
	h.ServeHTTP(loginRec, login)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", loginRec.Code, loginRec.Body.String())
	}
	cookie := loginRec.Result().Cookies()[0]

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password",
		strings.NewReader(`{"current_password":"wrong","new_password":"newpass"}`))
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong current password: %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/password",
		strings.NewReader(`{"current_password":"oldpass","new_password":"newpass"}`))
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("change password: %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"username":"root","password":"newpass"}`))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login with new password: %d", rec.Code)
	}
}
