package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/redis"
)

const redisRequestTimeout = 10 * time.Second

func listRedisKeysHandler(redisClient *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pattern := r.URL.Query().Get("pattern")
		ctx, cancel := context.WithTimeout(r.Context(), redisRequestTimeout)
		defer cancel()

		keys, err := redisClient.ListKeys(ctx, pattern)
		if err != nil {
			writeRedisError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, keys)
	}
}

func getRedisKeyHandler(redisClient *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		ctx, cancel := context.WithTimeout(r.Context(), redisRequestTimeout)
		defer cancel()

		detail, err := redisClient.GetKey(ctx, key)
		if err != nil {
			if errors.Is(err, redis.ErrNotFound) {
				writeJSONError(w, http.StatusNotFound, "not_found")
				return
			}
			writeRedisError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, detail)
	}
}

func deleteRedisKeyHandler(redisClient *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		ctx, cancel := context.WithTimeout(r.Context(), redisRequestTimeout)
		defer cancel()

		if err := redisClient.DeleteKey(ctx, key); err != nil {
			if errors.Is(err, redis.ErrNotFound) {
				writeJSONError(w, http.StatusNotFound, "not_found")
				return
			}
			writeRedisError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func writeRedisError(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{
		"error":   "redis_unavailable",
		"message": err.Error(),
	})
}
