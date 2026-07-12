package server

import (
	"context"
	"encoding/json"
	"net/http"
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
			InsecureSkipVerify: true,
		})
		if err != nil {
			return
		}

		ctx := r.Context()
		proc, err := terminal.Spawn("", nil, terminal.DefaultEnv(sess.Username), 80, 24)
		if err != nil {
			_ = writeTerminalError(conn, ctx, "spawn_failed")
			_ = conn.Close(websocket.StatusInternalError, "spawn_failed")
			return
		}

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			pumpPTYToWS(ctx, conn, proc.Master)
		}()

		go func() {
			defer wg.Done()
			pumpWSToPTY(ctx, conn, proc.Master)
		}()

		wg.Wait()

		if proc.Cmd != nil && proc.Cmd.Process != nil {
			_ = proc.Cmd.Process.Kill()
			_, _ = proc.Cmd.Process.Wait()
		}
		_ = proc.Master.Close()
		_ = conn.Close(websocket.StatusNormalClosure, "done")
	}
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

func writeTerminalError(conn *websocket.Conn, ctx context.Context, code string) error {
	data, err := json.Marshal(map[string]string{"type": "error", "error": code})
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, data)
}
