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

	TrainingJobTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeai_training_job_total",
			Help: "Total number of training jobs by status and framework",
		},
		[]string{"status", "framework"},
	)
	TrainingJobDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubeai_training_job_duration_seconds",
			Help:    "Duration of training jobs in seconds",
			Buckets: []float64{60, 300, 600, 1800, 3600, 7200, 14400, 28800, 86400},
		},
		[]string{"framework"},
	)
	TrainingJobGPUHour = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeai_training_job_gpu_hours_total",
			Help: "Total GPU hours consumed by training jobs",
		},
		[]string{"framework"},
	)
)

func init() {
	metrics.Registry.MustRegister(RequestTotal,
		RequestDuration, InferenceReplicas, TrainingJobDuration,
		TrainingJobGPUHour, TrainingJobTotal)
}
