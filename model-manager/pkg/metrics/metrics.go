package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ModelTotal 模型总数
	ModelTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "kubeai_model_total",
		Help: "Total number of models in the system",
	})

	// ModelVersionTotal 模型版本数 按状态分类统计
	ModelVersionTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kubeai_model_version_total",
		Help: "Total number of model versions in the system",
	}, []string{"status"})

	// ModelHealthStatus 模型健康状态 (1=healthy, 0=unhealthy)
	ModelHealthStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kubeai_model_health_status",
		Help: "Model health status (1=healthy, 0=unhealthy)",
	}, []string{"model_name", "version"})

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
