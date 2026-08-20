package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var (
	// RequestTotal API 请求总数 (Counter)
	RequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeai_inference_gateway_request_total",
			Help: "Total number of API requests",
		},
		[]string{"method", "path", "status", "service"},
	)

	// RequestDuration API 请求延迟
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubeai_inference_gateway_request_duration_seconds",
			Help:    "API request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "service"},
	)

	// InferenceReplicas 推理服务副本数
	InferenceReplicas = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubeai_inference_replicas",
			Help: "Ready replicas count for InferenceService (stable/canary version)",
		},
		[]string{"model_name", "model_version", "service"},
	)

	// InferenceTotal 推理请求总数
	InferenceTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeai_inference_total",
			Help: "Total number of inference requests",
		},
		[]string{"model_name", "model_version", "status"},
	)

	// InferenceDuration 推理执行时长
	InferenceDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubeai_inference_duration_seconds",
			Help:    "Inference execution duration in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"model_name", "model_version"},
	)

	// TrainingJobTotal 训练任务总数
	TrainingJobTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeai_training_job_total",
			Help: "Total number of training jobs by status and framework",
		},
		[]string{"status", "framework"},
	)

	// TrainingJobDuration 训练任务时长
	TrainingJobDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubeai_training_job_duration_seconds",
			Help:    "Duration of training jobs in seconds",
			Buckets: []float64{60, 300, 600, 1800, 3600, 7200, 14400, 28800, 86400},
		},
		[]string{"framework"},
	)

	// TrainingJobGPUHour 训练任务 GPU 小时数
	TrainingJobGPUHour = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeai_training_job_gpu_hours_total",
			Help: "Total GPU hours consumed by training jobs",
		},
		[]string{"framework"},
	)

	// QueueDepth 任务队列深度
	QueueDepth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubeai_inference_gateway_queue_depth",
			Help: "Current depth of task queues",
		},
		[]string{"queue_type"},
	)

	// DeadLetterTotal 死信队列消息总数
	DeadLetterTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeai_inference_gateway_dead_letter_total",
			Help: "Total number of messages in dead letter queue",
		},
		[]string{"queue_type", "reason"},
	)
)

func init() {
	// 注册所有指标
	prometheus.MustRegister(
		InferenceDuration,
		TrainingJobTotal,
		TrainingJobDuration,
		TrainingJobGPUHour,
	)
}

// Handler 返回 Prometheus metrics HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}
