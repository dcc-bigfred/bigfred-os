package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/auth"
)

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func changePasswordHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, ok := sessionFromContext(r.Context())
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var req changePasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_body")
			return
		}
		if req.CurrentPassword == "" || req.NewPassword == "" {
			writeJSONError(w, http.StatusBadRequest, "missing_password")
			return
		}

		if err := cfg.Auth.ChangePassword(sess.Username, req.CurrentPassword, req.NewPassword); err != nil {
			if errors.Is(err, auth.ErrInvalidCredentials) {
				writeJSONError(w, http.StatusUnauthorized, "invalid_credentials")
				return
			}
			if errors.Is(err, auth.ErrPasswordChangeFailed) {
				writeJSONError(w, http.StatusBadRequest, "password_change_failed")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
