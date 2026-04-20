// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package middleware

import (
	"kubeai-model-manager/pkg/metrics"
	"net/http"
	"strconv"
	"time"
)

type MetricsMiddleware struct {
}

func NewMetricsMiddleware() *MetricsMiddleware {
	return &MetricsMiddleware{}
}

func (m *MetricsMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := r.URL.Path
		method := r.Method

		// 使用自定义包装器
		wrapped := &responseWriter{
			ResponseWriter: w,
			status:         http.StatusOK, // 默认状态码
		}
		next(wrapped, r)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(wrapped.status)

		metrics.RequestTotal.WithLabelValues(method, path, status, "model-manager").Inc()
		metrics.RequestDuration.WithLabelValues(method, path, "model-manager").Observe(duration)
	}
}

// 自定义 ResponseWriter 包装器
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
