package server

import (
	"net/http"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/logs"
)

func listLogsHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries, err := logs.ListAll(cfg.LogRoots)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if entries == nil {
			entries = []logs.Entry{}
		}
		writeJSON(w, http.StatusOK, entries)
	}
}
