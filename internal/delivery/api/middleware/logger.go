package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
)

func StructuredLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()

		defer func() {
			duration := time.Since(start)

			logger.InfoContext(r.Context(), "http request",
				"http.request.method", r.Method,
				"http.route", r.URL.Path,
				"http.request.query", r.URL.RawQuery,
				"http.response.status_code", ww.Status(),
				"http.response.bytes", ww.BytesWritten(),
				"http.server.request.duration_ms", duration.Milliseconds(),
				"http.request.remote_addr", r.RemoteAddr,
				"request_id", middleware.GetReqID(r.Context()),
			)
		}()

		next.ServeHTTP(ww, r)
	})
}
