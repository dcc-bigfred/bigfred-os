package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/etcdir"
)

func listEtcFilesHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := etcdir.List(cfg.EtcDir)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func readEtcFileHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		body, err := etcdir.Read(cfg.EtcDir, path)
		if err != nil {
			writeEtcFileError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, body)
	}
}

type etcFileWriteRequest struct {
	Content string `json:"content"`
}

func writeEtcFileHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		var req etcFileWriteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_body")
			return
		}

		body, err := etcdir.Write(cfg.EtcDir, path, req.Content)
		if err != nil {
			writeEtcFileError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, body)
	}
}

func writeEtcFileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, etcdir.ErrNotFound):
		writeJSONError(w, http.StatusNotFound, "not_found")
	case errors.Is(err, etcdir.ErrInvalidPath):
		writeJSONError(w, http.StatusBadRequest, "invalid_path")
	case errors.Is(err, etcdir.ErrTooLarge):
		writeJSONError(w, http.StatusRequestEntityTooLarge, "too_large")
	case errors.Is(err, etcdir.ErrNotFile):
		writeJSONError(w, http.StatusBadRequest, "not_a_file")
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "write_failed",
			"message": err.Error(),
		})
	}
}
