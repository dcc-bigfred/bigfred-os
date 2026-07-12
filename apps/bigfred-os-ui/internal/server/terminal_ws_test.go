package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/auth"
	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/terminal"
)

func TestStreamTerminalHandler(t *testing.T) {
	probe, err := terminal.Spawn("/bin/sh", []string{"-c", "true"}, nil, 80, 24)
	if err != nil {
		t.Skipf("PTY not available: %v", err)
	}
	_ = probe.Cmd.Process.Kill()
	_ = probe.Master.Close()

	authSvc, err := auth.NewStatic("admin", "secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	h := NewRouter(Config{Auth: authSvc})

	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"admin","password":"secret"}`))
	loginRec := httptest.NewRecorder()
	h.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status: %d %s", loginRec.Code, loginRec.Body.String())
	}
	cookie := loginRec.Result().Cookies()[0]

	srv := httptest.NewServer(h)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/terminal"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Cookie": {cookie.Name + "=" + cookie.Value},
		},
	})
	if err != nil {
		t.Fatalf("dial: %v (status %v)", err, resp)
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")

	var output strings.Builder
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			typ, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			if typ == websocket.MessageBinary {
				output.Write(data)
			}
		}
	}()

	waitForOutput := func(timeout time.Duration) bool {
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			if strings.Contains(output.String(), "test") {
				return true
			}
			time.Sleep(20 * time.Millisecond)
		}
		return strings.Contains(output.String(), "test")
	}

	time.Sleep(100 * time.Millisecond)

	resize, err := json.Marshal(map[string]any{
		"type": "resize",
		"cols": 120,
		"rows": 40,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Write(ctx, websocket.MessageText, resize); err != nil {
		t.Fatal(err)
	}

	if err := conn.Write(ctx, websocket.MessageBinary, []byte("echo test\n")); err != nil {
		t.Fatal(err)
	}

	if !waitForOutput(5 * time.Second) {
		t.Fatalf("expected terminal output to contain test, got %q", output.String())
	}
}

func TestStreamTerminalUnauthorized(t *testing.T) {
	authSvc, err := auth.NewStatic("admin", "secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	h := NewRouter(Config{Auth: authSvc})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/terminal", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: %d", rec.Code)
	}
}
