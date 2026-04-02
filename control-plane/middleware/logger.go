package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// StructuredLogger returns middleware that logs each request using slog
// in JSON format: method, path, status, duration_ms, ip, and user_id
// (when the request context contains an authenticated user).
func StructuredLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := newResponseWriter(w)

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		attrs := []slog.Attr{
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", wrapped.statusCode),
			slog.Float64("duration_ms", float64(duration.Microseconds())/1000.0),
			slog.String("ip", r.RemoteAddr),
		}

		// Add user_id if the request was authenticated
		if uid, ok := r.Context().Value(UserIDKey).(int64); ok && uid != 0 {
			attrs = append(attrs, slog.Int64("user_id", uid))
		}

		slog.LogAttrs(r.Context(), slog.LevelInfo, "http_request", attrs...)
	})
}
