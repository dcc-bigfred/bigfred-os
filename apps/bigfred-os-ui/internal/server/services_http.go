package server

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/services"
)

func listServicesHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := services.List(cfg.InitDir)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
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
		if err := services.Control(cfg.InitDir, id, action); err != nil {
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
