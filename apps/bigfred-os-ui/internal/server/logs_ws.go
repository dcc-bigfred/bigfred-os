package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/coder/websocket"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/logs"
)

const historyLines = 300

type wsMessage struct {
	Type  string   `json:"type"`
	Lines []string `json:"lines"`
	Text  string   `json:"text,omitempty"`
	Error string   `json:"error,omitempty"`
}

func streamLogsHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := sessionToken(r)
		if token == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if _, err := cfg.Auth.VerifyToken(token); err != nil {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		logID := r.URL.Query().Get("id")
		if logID == "" {
			writeJSONError(w, http.StatusBadRequest, "missing_id")
			return
		}

		path, err := logs.ResolvePath(cfg.LogRoots, logID)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_id")
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "done")

		ctx := r.Context()
		if err := sendHistory(ctx, conn, path); err != nil {
			_ = writeWS(conn, ctx, wsMessage{Type: "error", Error: "read_failed"})
			return
		}

		size, err := logs.FileSize(path)
		if err != nil {
			_ = writeWS(conn, ctx, wsMessage{Type: "error", Error: "read_failed"})
			return
		}

		tailer := logs.NewTailer(path, size)
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				lines, _, err := tailer.ReadNew()
				if err != nil {
					if os.IsNotExist(err) {
						_ = writeWS(conn, ctx, wsMessage{Type: "error", Error: "file_gone"})
						return
					}
					continue
				}
				for _, line := range lines {
					if err := writeWS(conn, ctx, wsMessage{Type: "line", Text: line}); err != nil {
						return
					}
				}
			}
		}
	}
}

func sendHistory(ctx context.Context, conn *websocket.Conn, path string) error {
	lines, err := logs.TailLast(path, historyLines)
	if err != nil {
		return err
	}
	if lines == nil {
		lines = []string{}
	}
	return writeWS(conn, ctx, wsMessage{Type: "history", Lines: lines})
}

func writeWS(conn *websocket.Conn, ctx context.Context, msg wsMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, data)
}
