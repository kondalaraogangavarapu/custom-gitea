package middleware

import (
	"context"
	"log"
	"net/http"
	"time"
)

type contextKey string

const UserIDKey contextKey = "user_id"

// SessionStore is the interface the middleware needs to validate sessions.
type SessionStore interface {
	GetSession(id string) (int64, error)
}

// NewSessionAuth creates a session authentication middleware.
// When oidcEnabled is true, unauthenticated requests to pages get redirected to
// /auth/login-page, and API requests get a 401. When OIDC is disabled, all
// requests fall through as the default admin (ID=1) for backward compatibility.
func NewSessionAuth(store SessionStore, oidcEnabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to read session from cookie
			cookie, err := r.Cookie("aetherdev_session")
			if err == nil && cookie.Value != "" {
				userID, err := store.GetSession(cookie.Value)
				if err == nil && userID > 0 {
					ctx := context.WithValue(r.Context(), UserIDKey, userID)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			if !oidcEnabled {
				// OIDC not configured — fall back to default admin user
				ctx := context.WithValue(r.Context(), UserIDKey, int64(1))
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// No valid session and OIDC is enabled — require auth
			if isAPIRequest(r) {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// Page request — redirect to login page
			http.Redirect(w, r, "/auth/login-page", http.StatusFound)
		})
	}
}

func isAPIRequest(r *http.Request) bool {
	return len(r.URL.Path) >= 8 && r.URL.Path[:8] == "/api/v1/"
}

// Logger logs request method, path, status, and duration.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
	})
}

// Recovery catches panics and returns a 500.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// CORS adds basic CORS headers for API access.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
