package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var (
	RequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeai_api_gateway_request_total",
			Help: "Total number of API requests",
		},
		[]string{"method", "path", "status", "service"},
	)
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubeai_api_gateway_request_duration_seconds",
			Help:    "API request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "service"},
	)
)

// Handler 暴露Prometheus指标接口
func Handler() http.Handler {
	return promhttp.Handler()
}
