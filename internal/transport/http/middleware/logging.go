package middleware

import (
	"log/slog"
	"net/http"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// RequestLogger 输出结构化请求日志。
//
// 用 slog 而不是 fmt.Println，是为了让日志能被容器直接收集、按字段检索。
func RequestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				log.Info("http request",
					"method", r.Method,
					"path", r.URL.Path,
					"status", ww.Status(),
					"duration_ms", time.Since(start).Milliseconds(),
					"bytes", ww.BytesWritten(),
					"request_id", chimw.GetReqID(r.Context()),
					"remote_ip", r.RemoteAddr,
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
