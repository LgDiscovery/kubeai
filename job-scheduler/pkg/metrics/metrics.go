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
			Name: "kubeai_job_scheduler_request_total",
			Help: "Total number of API requests",
		},
		[]string{"method", "path", "status", "service"},
	)

	// RequestDuration API 请求延迟
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubeai_job_scheduler_request_duration_seconds",
			Help:    "API request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "service"},
	)

	// TaskTotal 任务总数（按类型和状态）
	TaskTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeai_job_scheduler_task_total",
			Help: "Total number of tasks by type and status",
		},
		[]string{"task_type", "status"},
	)

	// TaskDuration 任务执行时长
	TaskDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubeai_job_scheduler_task_duration_seconds",
			Help:    "Task execution duration in seconds",
			Buckets: []float64{10, 30, 60, 120, 300, 600, 1800, 3600, 7200, 14400, 28800, 86400},
		},
		[]string{"task_type", "framework"},
	)

	// QueueDepth 任务队列深度
	QueueDepth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubeai_job_scheduler_queue_depth",
			Help: "Current depth of task queues",
		},
		[]string{"queue_type"},
	)

	// TaskRetryTotal 任务重试总数
	TaskRetryTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeai_job_scheduler_task_retry_total",
			Help: "Total number of task retries",
		},
		[]string{"task_type", "reason"},
	)

	// TaskFailedTotal 任务失败总数
	TaskFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeai_job_scheduler_task_failed_total",
			Help: "Total number of failed tasks",
		},
		[]string{"task_type", "error_type"},
	)

	// DBConnectionOpen 数据库打开连接数
	DBConnectionOpen = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "kubeai_job_scheduler_db_connections_open",
		Help: "Number of open database connections",
	})
)

// Handler 返回 Prometheus metrics HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}
