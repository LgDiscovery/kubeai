// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package middleware

import (
	"kubeai-api-gateway/pkg/metrics"
	"net/http"
	"strconv"
	"strings"
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

		service := "unknown"

		// 自动识别服务分组（精准标记 /inference 接口）
		if strings.HasPrefix(path, "/api/v1/inference") {
			service = "inference" // 推理接口专用标识
		} else if strings.HasPrefix(path, "/api/v1/models") {
			service = "models"
		} else if strings.HasPrefix(path, "/api/v1/jobs") {
			service = "jobs"
		} else if strings.HasPrefix(path, "/api/v1/observer") {
			service = "observer"
		}

		metrics.RequestTotal.WithLabelValues(method, path, status, service).Inc()
		metrics.RequestDuration.WithLabelValues(method, path, service).Observe(duration)
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
