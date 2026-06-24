package auth

import (
	"testing"
	"time"
)

func TestLoginAndVerify(t *testing.T) {
	svc, err := NewStatic("admin", "secret", time.Hour)
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

func TestStaticChangePassword(t *testing.T) {
	svc, err := NewStatic("root", "old", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ChangePassword("root", "wrong", "new"); err != ErrInvalidCredentials {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
	if err := svc.ChangePassword("root", "old", "new"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Login("root", "new"); err != nil {
		t.Fatalf("login with new password: %v", err)
	}
}
