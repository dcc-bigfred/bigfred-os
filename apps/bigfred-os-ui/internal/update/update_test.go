package update_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/update"
)

func TestApplyBigFred(t *testing.T) {
	payload := []byte("#!/bin/sh\necho bigfred\n")

	var apiHits, dlHits int
	mux := http.NewServeMux()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)

	mux.HandleFunc("/repos/dcc-bigfred/bigfred/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		apiHits++
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("missing auth header: %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{
			"tag_name": "v1.2.3",
			"assets": [
				{"name": "loco-server-linux-arm64", "browser_download_url": %q, "size": 12}
			]
		}`, srv.URL+"/download/loco-server")
	})
	mux.HandleFunc("/download/loco-server", func(w http.ResponseWriter, _ *http.Request) {
		dlHits++
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(payload)
	})

	client := srv.Client()
	origTransport := client.Transport
	if origTransport == nil {
		origTransport = http.DefaultTransport
	}
	client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if strings.HasPrefix(req.URL.String(), "https://api.github.com/repos/") {
			req = req.Clone(req.Context())
			req.URL.Scheme = "http"
			req.URL.Host = strings.TrimPrefix(srv.URL, "http://")
			req.URL.Path = "/repos/dcc-bigfred/bigfred/releases/latest"
			req.Host = req.URL.Host
		}
		return origTransport.RoundTrip(req)
	})

	dir := t.TempDir()
	u := update.New(update.Config{
		InstallDir:  dir,
		Arch:        "arm64",
		HTTPClient:  client,
		GitHubToken: "test-token",
	})

	res, err := u.Apply(context.Background(), update.TargetBigFred, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Tag != "v1.2.3" || res.Path != filepath.Join(dir, "bigfred") || res.Restart != "bigfred" {
		t.Fatalf("result: %+v", res)
	}
	got, err := os.ReadFile(res.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("installed payload mismatch")
	}
	fi, err := os.Stat(res.Path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm()&0o111 == 0 {
		t.Fatalf("not executable: %v", fi.Mode())
	}
	if apiHits != 1 || dlHits != 1 {
		t.Fatalf("hits api=%d dl=%d", apiHits, dlHits)
	}
}

func TestParseTarget(t *testing.T) {
	cases := map[string]update.Target{
		"bigfred":       update.TargetBigFred,
		"bigfred-ui":    update.TargetBigFredUI,
		"bigfred-os-ui": update.TargetBigFredUI,
	}
	for in, want := range cases {
		got, err := update.ParseTarget(in)
		if err != nil || got != want {
			t.Fatalf("%q: got %q err %v", in, got, err)
		}
	}
	if _, err := update.ParseTarget("nope"); err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyMissingAsset(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	mux.HandleFunc("/repos/dcc-bigfred/bigfred-os/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"tag_name":"v9","assets":[{"name":"other","browser_download_url":"http://x"}]}`)
	})

	client := srv.Client()
	orig := client.Transport
	if orig == nil {
		orig = http.DefaultTransport
	}
	client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req = req.Clone(req.Context())
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		req.URL.Path = "/repos/dcc-bigfred/bigfred-os/releases/latest"
		req.Host = req.URL.Host
		return orig.RoundTrip(req)
	})

	u := update.New(update.Config{
		InstallDir: t.TempDir(),
		Arch:       "arm64",
		HTTPClient: client,
	})
	_, err := u.Apply(context.Background(), update.TargetBigFredUI, "")
	if err == nil || !strings.Contains(err.Error(), "no matching release asset") {
		t.Fatalf("err=%v", err)
	}
}

func TestListReleases(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	mux.HandleFunc("/repos/dcc-bigfred/bigfred/releases", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[
			{"tag_name":"v2","name":"Two","draft":false,"prerelease":false,"published_at":"2026-01-02T00:00:00Z",
			 "assets":[{"name":"loco-server-linux-arm64","browser_download_url":"http://x","size":1}]},
			{"tag_name":"v1","name":"One","draft":false,"prerelease":true,"published_at":"2026-01-01T00:00:00Z",
			 "assets":[{"name":"other","browser_download_url":"http://x","size":1}]},
			{"tag_name":"v0","draft":true,"assets":[{"name":"loco-server-linux-arm64","browser_download_url":"http://x","size":1}]}
		]`)
	})

	client := srv.Client()
	orig := client.Transport
	if orig == nil {
		orig = http.DefaultTransport
	}
	client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req = req.Clone(req.Context())
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		req.URL.Path = "/repos/dcc-bigfred/bigfred/releases"
		req.URL.RawQuery = ""
		req.Host = req.URL.Host
		return orig.RoundTrip(req)
	})

	u := update.New(update.Config{
		InstallDir: t.TempDir(),
		Arch:       "arm64",
		HTTPClient: client,
	})
	list, err := u.ListReleases(context.Background(), update.TargetBigFred)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Tag != "v2" {
		t.Fatalf("list=%+v", list)
	}
}

func TestApplyByTag(t *testing.T) {
	payload := []byte("tagged")
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	mux.HandleFunc("/repos/dcc-bigfred/bigfred/releases/tags/v9.9.9", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, `{
			"tag_name":"v9.9.9",
			"assets":[{"name":"loco-server-linux-arm64","browser_download_url":%q,"size":6}]
		}`, srv.URL+"/bin")
	})
	mux.HandleFunc("/bin", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(payload)
	})

	client := srv.Client()
	orig := client.Transport
	if orig == nil {
		orig = http.DefaultTransport
	}
	client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "/releases/tags/") {
			req = req.Clone(req.Context())
			req.URL.Scheme = "http"
			req.URL.Host = strings.TrimPrefix(srv.URL, "http://")
			req.URL.Path = "/repos/dcc-bigfred/bigfred/releases/tags/v9.9.9"
			req.Host = req.URL.Host
		}
		return orig.RoundTrip(req)
	})

	dir := t.TempDir()
	u := update.New(update.Config{
		InstallDir: dir,
		Arch:       "arm64",
		HTTPClient: client,
	})
	res, err := u.Apply(context.Background(), update.TargetBigFred, "v9.9.9")
	if err != nil {
		t.Fatal(err)
	}
	if res.Tag != "v9.9.9" {
		t.Fatalf("tag=%s", res.Tag)
	}
	got, _ := os.ReadFile(res.Path)
	if string(got) != string(payload) {
		t.Fatalf("got=%q", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
