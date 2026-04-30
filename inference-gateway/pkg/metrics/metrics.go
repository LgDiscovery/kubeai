package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// RequestTotal API 请求总数
	RequestTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kubeai_inference_request_total",
		Help: "Total number of API requests",
	}, []string{"method", "path", "status", "service"})

	// RequestDuration API 请求延迟
	RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "kubeai_inference_request_latency",
		Help:    "Latency of API requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "service"})

	InferenceReplicas = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubeai_inference_replicas",                                         // 指标名称（你指定的）
			Help: "Ready replicas count for InferenceService (stable/canary version)", // 指标说明（必填，用于监控文档）
		},
		// 👇 关键：标签列表，顺序必须和 WithLabelValues 完全一致
		[]string{"model_name", "model_version", "service"})
)

func init() {
	metrics.Registry.MustRegister(RequestTotal, RequestDuration, InferenceReplicas)
}
