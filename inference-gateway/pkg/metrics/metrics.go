package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RequestTotal API 请求总数
	RequestTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kubeai_api_request_total",
		Help: "Total number of API requests",
	}, []string{"method", "path", "status", "service"})

	// RequestDuration API 请求延迟
	RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "kubeai_api_request_latency",
		Help:    "Latency of API requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "service"})
)
