package server

import (
	"errors"
	"io"
	"net/http"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"

	miclient "github.com/dcc-bigfred/microinit/go/client"
	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/services"
)

func listServicesHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := services.List(cfg.Microinit)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{
				"error":   "microinit_unavailable",
				"message": err.Error(),
			})
			return
		}
		if list == nil {
			list = []services.Service{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func serviceActionHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		action := chi.URLParam(r, "action")
		if err := services.Control(cfg.Microinit, id, action); err != nil {
			switch {
			case errors.Is(err, services.ErrInvalidID),
				errors.Is(err, services.ErrInvalidAction),
				errors.Is(err, services.ErrNotFound):
				writeJSONError(w, http.StatusBadRequest, "bad_request")
			default:
				writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
					"error":   "action_failed",
					"message": err.Error(),
				})
			}
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func streamServiceLogsHandler(cfg Config) http.HandlerFunc {
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

		id := chi.URLParam(r, "id")
		if err := miclient.ValidateName(id); err != nil {
			writeJSONError(w, http.StatusBadRequest, "bad_request")
			return
		}
		if cfg.MicroinitClient == nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{
				"error":   "microinit_unavailable",
				"message": "log streaming requires microinit client",
			})
			return
		}

		wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			return
		}
		defer wsConn.Close(websocket.StatusNormalClosure, "done")

		ctx := r.Context()

		history, err := fetchServiceLogHistory(cfg.MicroinitClient, id, historyLines)
		if err != nil {
			_ = writeWS(wsConn, ctx, wsMessage{Type: "error", Error: err.Error()})
			return
		}
		if err := writeWS(wsConn, ctx, wsMessage{Type: "history", Lines: history}); err != nil {
			return
		}

		unix, err := cfg.MicroinitClient.FollowLogs(id, 0, true)
		if err != nil {
			_ = writeWS(wsConn, ctx, wsMessage{Type: "error", Error: err.Error()})
			return
		}
		defer unix.Close()

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			resp, err := cfg.MicroinitClient.ReadFrame(unix)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				_ = writeWS(wsConn, ctx, wsMessage{Type: "error", Error: "stream_failed"})
				return
			}

			switch resp.Type {
			case "log":
				if resp.Line == nil {
					continue
				}
				text := miclient.FormatLogLine(*resp.Line)
				if err := writeWS(wsConn, ctx, wsMessage{Type: "line", Text: text}); err != nil {
					return
				}
			case "error":
				msg := resp.Message
				if msg == "" {
					msg = "stream_failed"
				}
				_ = writeWS(wsConn, ctx, wsMessage{Type: "error", Error: msg})
				return
			case "ok":
				return
			}
		}
	}
}

func fetchServiceLogHistory(client *miclient.Client, name string, lines int) ([]string, error) {
	unix, err := client.FollowLogs(name, lines, false)
	if err != nil {
		return nil, err
	}
	defer unix.Close()

	out := make([]string, 0, 64)
	for {
		resp, err := client.ReadFrame(unix)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return out, nil
			}
			return nil, err
		}
		switch resp.Type {
		case "log":
			if resp.Line != nil {
				out = append(out, miclient.FormatLogLine(*resp.Line))
			}
		case "ok":
			return out, nil
		case "error":
			msg := resp.Message
			if msg == "" {
				msg = "read_failed"
			}
			return nil, errors.New(msg)
		}
	}
}
