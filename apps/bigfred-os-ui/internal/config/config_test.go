package config

import "testing"

func TestParseDotenv(t *testing.T) {
	f := Parse(`# hub admin UI
HTTP=0.0.0.0:9090
USERNAME=ops
PASSWORD="s3cret"
LOG_ROOT=/var/log/hub
SECURE_COOKIE=true
`)
	if f.HTTP != "0.0.0.0:9090" {
		t.Fatalf("http: %q", f.HTTP)
	}
	if f.Username != "ops" || f.Password != "s3cret" {
		t.Fatalf("credentials: %+v", f)
	}
	if f.LogRoot != "/var/log/hub" {
		t.Fatalf("log root: %q", f.LogRoot)
	}
	if f.SecureCookie == nil || !*f.SecureCookie {
		t.Fatal("expected secure cookie true")
	}
}
