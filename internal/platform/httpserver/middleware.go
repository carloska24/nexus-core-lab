package httpserver

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

type contextKey string

const (
	RequestIDHeader                = "X-Request-ID"
	RequestIDContextKey contextKey = "request_id"
)

// GetRequestID recupera o Request ID associado ao contexto da requisição.
func GetRequestID(ctx context.Context) string {
	if val, ok := ctx.Value(RequestIDContextKey).(string); ok {
		return val
	}
	return ""
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// TelemetryMiddleware intercepta as requisições HTTP gerenciando o cabeçalho X-Request-ID,
// registrando logs estruturados via log/slog e contabilizando as requisições atômicas.
func TelemetryMiddleware(requestsCounter *atomic.Uint64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			reqID := r.Header.Get(RequestIDHeader)
			if reqID == "" {
				reqID = generateRequestID()
			}

			w.Header().Set(RequestIDHeader, reqID)
			ctx := context.WithValue(r.Context(), RequestIDContextKey, reqID)

			if requestsCounter != nil {
				requestsCounter.Add(1)
			}

			sw := &statusResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // padrão HTTP
			}

			next.ServeHTTP(sw, r.WithContext(ctx))

			duration := time.Since(start)

			slog.InfoContext(ctx, "http_request",
				slog.String("request_id", reqID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", sw.statusCode),
				slog.Int64("duration_ms", duration.Milliseconds()),
			)
		})
	}
}

func generateRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16],
	)
}
