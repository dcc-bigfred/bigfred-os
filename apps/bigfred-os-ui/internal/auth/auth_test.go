package auth

import (
	"testing"
	"time"
)

func TestLoginAndVerify(t *testing.T) {
	svc, err := New("admin", "secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Login("admin", "wrong"); err != ErrInvalidCredentials {
		t.Fatalf("expected invalid credentials, got %v", err)
	}

	sess, err := svc.Login("admin", "secret")
	if err != nil {
		t.Fatal(err)
	}

	token, _, err := svc.IssueToken(sess)
	if err != nil {
		t.Fatal(err)
	}

	got, err := svc.VerifyToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if got.Username != "admin" {
		t.Fatalf("username: got %q", got.Username)
	}
}

func TestNewRequiresCredentials(t *testing.T) {
	if _, err := New("", "x", time.Hour); err == nil {
		t.Fatal("expected error for empty username")
	}
}
