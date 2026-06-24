//go:build !pam

package auth

import (
	"testing"
	"time"
)

func TestNewRequiresCredentialsWithoutPAM(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("expected error for empty config without pam tag")
	}
	if _, err := New(Config{PAMService: "bigfred-os-ui"}); err == nil {
		t.Fatal("expected error when only PAM is configured without pam tag")
	}
}

func TestNewStaticIgnoresPAMServiceWithoutPAMTag(t *testing.T) {
	svc, err := New(Config{
		PAMService: "bigfred-os-ui",
		Username:   "admin",
		Password:   "secret",
		TTL:        time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Login("admin", "secret"); err != nil {
		t.Fatal(err)
	}
}
