package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/coder/websocket"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/terminal"
)

type terminalControlMessage struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

func streamTerminalHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := sessionToken(r)
		if token == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		sess, err := cfg.Auth.VerifyToken(token)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: terminalOriginPatterns(cfg.DevOrigins),
		})
		if err != nil {
			return
		}

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		proc, err := terminal.Spawn("", nil, terminal.DefaultEnv(sess.Username), 80, 24)
		if err != nil {
			_ = writeTerminalError(ctx, conn, "spawn_failed")
			_ = conn.Close(websocket.StatusInternalError, "spawn_failed")
			return
		}

		log.Printf("terminal: session opened for user %q", sess.Username)
		defer log.Printf("terminal: session closed for user %q", sess.Username)

		var once sync.Once
		stop := func() {
			once.Do(func() {
				cancel()
				_ = proc.Master.Close()
			})
		}

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			defer stop()
			pumpPTYToWS(ctx, conn, proc.Master)
		}()

		go func() {
			defer wg.Done()
			defer stop()
			pumpWSToPTY(ctx, conn, proc.Master)
		}()

		wg.Wait()

		if proc.Cmd != nil && proc.Cmd.Process != nil {
			_ = proc.Cmd.Process.Kill()
			_, _ = proc.Cmd.Process.Wait()
		}
		_ = conn.Close(websocket.StatusNormalClosure, "done")
	}
}

func terminalOriginPatterns(devOrigins []string) []string {
	patterns := make([]string, 0, len(devOrigins))
	for _, o := range devOrigins {
		u, err := url.Parse(o)
		if err != nil || u.Host == "" {
			continue
		}
		patterns = append(patterns, u.Host)
	}
	return patterns
}

func pumpPTYToWS(ctx context.Context, conn *websocket.Conn, master *os.File) {
	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := master.Read(buf)
		if n > 0 {
			if writeErr := conn.Write(ctx, websocket.MessageBinary, buf[:n]); writeErr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

func pumpWSToPTY(ctx context.Context, conn *websocket.Conn, master *os.File) {
	for {
		typ, payload, err := conn.Read(ctx)
		if err != nil {
			return
		}

		switch typ {
		case websocket.MessageBinary:
			if len(payload) == 0 {
				continue
			}
			if _, err := master.Write(payload); err != nil {
				return
			}
		case websocket.MessageText:
			var msg terminalControlMessage
			if err := json.Unmarshal(payload, &msg); err != nil {
				continue
			}
			if msg.Type == "resize" && msg.Cols > 0 && msg.Rows > 0 {
				_ = terminal.Resize(master, msg.Cols, msg.Rows)
			}
		}
	}
}

func writeTerminalError(ctx context.Context, conn *websocket.Conn, code string) error {
	data, err := json.Marshal(map[string]string{"type": "error", "error": code})
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, data)
}
