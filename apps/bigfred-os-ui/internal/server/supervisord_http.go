package server

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/supervisord"
)

func listSupervisordProgramsHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := supervisord.List(cfg.SupervisordConf)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if list == nil {
			list = []supervisord.Program{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func supervisordProgramActionHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		action := chi.URLParam(r, "action")
		if err := supervisord.Control(cfg.SupervisordConf, name, action); err != nil {
			switch {
			case errors.Is(err, supervisord.ErrInvalidName),
				errors.Is(err, supervisord.ErrInvalidAction),
				errors.Is(err, supervisord.ErrNotFound):
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
