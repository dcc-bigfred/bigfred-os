package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/update"
)

func listUpdateReleasesHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.Updater == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "updates_disabled")
			return
		}
		target, err := update.ParseTarget(chi.URLParam(r, "target"))
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "unknown_target")
			return
		}

		list, err := cfg.Updater.ListReleases(r.Context(), target)
		if err != nil {
			writeUpdateError(w, err)
			return
		}
		if list == nil {
			list = []update.Release{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

type updateRequest struct {
	Tag string `json:"tag"`
}

func runUpdateHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.Updater == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "updates_disabled")
			return
		}
		target, err := update.ParseTarget(chi.URLParam(r, "target"))
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "unknown_target")
			return
		}

		var req updateRequest
		if r.Body != nil && r.ContentLength != 0 {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid_body")
				return
			}
		}
		tag := strings.TrimSpace(req.Tag)
		if tag == "" {
			tag = strings.TrimSpace(r.URL.Query().Get("tag"))
		}

		res, err := cfg.Updater.Apply(r.Context(), target, tag)
		if err != nil {
			writeUpdateError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, res)
	}
}

func writeUpdateError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, update.ErrUnknownTarget):
		writeJSONError(w, http.StatusBadRequest, "unknown_target")
	case errors.Is(err, update.ErrNoRelease):
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":   "no_release",
			"message": err.Error(),
		})
	case errors.Is(err, update.ErrNoAsset):
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":   "no_asset",
			"message": err.Error(),
		})
	default:
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"error":   "update_failed",
			"message": err.Error(),
		})
	}
}
