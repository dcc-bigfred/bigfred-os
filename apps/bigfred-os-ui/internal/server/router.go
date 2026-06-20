package server

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/auth"
)

// Config holds runtime dependencies for the HTTP server.
type Config struct {
	Auth         *auth.Service
	LogRoots     []string
	InitDir      string
	StaticFS     fs.FS
	SecureCookie bool
	DevOrigins   []string
}

func NewRouter(cfg Config) http.Handler {
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	origins := []string{}
	if len(cfg.DevOrigins) > 0 {
		origins = cfg.DevOrigins
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", loginHandler(cfg))
		r.Post("/auth/logout", logoutHandler(cfg))

		r.Group(func(r chi.Router) {
			r.Use(requireAuth(cfg.Auth))
			r.Get("/auth/me", meHandler(cfg))
			r.Get("/logs", listLogsHandler(cfg))
			r.Get("/logs/stream", streamLogsHandler(cfg))
			r.Get("/services", listServicesHandler(cfg))
			r.Post("/services/{id}/{action}", serviceActionHandler(cfg))
		})
	})

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	if cfg.StaticFS != nil {
		h := spaHandler(cfg.StaticFS)
		r.Get("/", h.ServeHTTP)
		r.Get("/*", h.ServeHTTP)
	}

	return r
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type meResponse struct {
	Username string `json:"username"`
}

func loginHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_body")
			return
		}
		req.Username = strings.TrimSpace(req.Username)
		if req.Username == "" || req.Password == "" {
			writeJSONError(w, http.StatusBadRequest, "missing_credentials")
			return
		}

		sess, err := cfg.Auth.Login(req.Username, req.Password)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidCredentials) {
				writeJSONError(w, http.StatusUnauthorized, "invalid_credentials")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		token, expiry, err := cfg.Auth.IssueToken(sess)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     auth.SessionCookieName,
			Value:    token,
			Path:     "/",
			Expires:  expiry,
			MaxAge:   int(cfg.Auth.SessionTTL().Seconds()),
			HttpOnly: true,
			Secure:   cfg.SecureCookie,
			SameSite: http.SameSiteLaxMode,
		})

		writeJSON(w, http.StatusOK, meResponse{Username: sess.Username})
	}
}

func logoutHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     auth.SessionCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   cfg.SecureCookie,
			SameSite: http.SameSiteLaxMode,
		})
		w.WriteHeader(http.StatusNoContent)
	}
}

func meHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, ok := sessionFromContext(r.Context())
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		writeJSON(w, http.StatusOK, meResponse{Username: sess.Username})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}

func sessionToken(r *http.Request) string {
	if c, err := r.Cookie(auth.SessionCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	return r.URL.Query().Get("token")
}

func requireAuth(svc *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := sessionToken(r)
			if token == "" {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			sess, err := svc.VerifyToken(token)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			next.ServeHTTP(w, r.WithContext(withSession(r.Context(), sess)))
		})
	}
}

// sleep exported for tests
var pollInterval = 400 * time.Millisecond
