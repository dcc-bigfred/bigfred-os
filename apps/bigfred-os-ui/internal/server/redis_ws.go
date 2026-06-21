package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/coder/websocket"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/redis"
)

type redisWSMessage struct {
	Type   string           `json:"type"`
	Detail *redis.KeyDetail `json:"detail,omitempty"`
	Error  string           `json:"error,omitempty"`
}

func streamRedisKeyHandler(cfg Config, redisClient *redis.Client) http.HandlerFunc {
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

		key := r.URL.Query().Get("key")
		if key == "" {
			writeJSONError(w, http.StatusBadRequest, "missing_key")
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
		err = redisClient.WatchKey(ctx, key, func(ev redis.WatchEvent) {
			switch ev.Kind {
			case redis.WatchSnapshot:
				_ = writeRedisWS(conn, ctx, redisWSMessage{Type: "snapshot", Detail: &ev.Detail})
			case redis.WatchUpdate:
				_ = writeRedisWS(conn, ctx, redisWSMessage{Type: "update", Detail: &ev.Detail})
			case redis.WatchDeleted:
				_ = writeRedisWS(conn, ctx, redisWSMessage{Type: "deleted"})
			}
		})
		if err != nil && ctx.Err() == nil {
			_ = writeRedisWS(conn, ctx, redisWSMessage{Type: "error", Error: "watch_failed"})
		}
	}
}

func writeRedisWS(conn *websocket.Conn, ctx context.Context, msg redisWSMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, data)
}
