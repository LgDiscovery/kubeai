package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var (
	// RequestTotal API 请求总数 (Counter，只增不减)
	RequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeai_model_manager_request_total",
			Help: "Total number of API requests",
		},
		[]string{"method", "path", "status", "service"},
	)

	// RequestDuration API 请求延迟
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubeai_model_manager_request_duration_seconds",
			Help:    "API request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "service"},
	)

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

	// ModelUploadTotal 模型文件上传总数
	ModelUploadTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeai_model_upload_total",
			Help: "Total number of model file uploads",
		},
		[]string{"model_name", "version", "status"},
	)

	// ModelUploadDuration 模型文件上传延迟
	ModelUploadDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubeai_model_upload_duration_seconds",
			Help:    "Model file upload duration in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600},
		},
		[]string{"model_name", "version"},
	)

	// ModelDownloadTotal 模型文件下载总数
	ModelDownloadTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeai_model_download_total",
			Help: "Total number of model file downloads",
		},
		[]string{"model_name", "version", "status"},
	)

	// DBConnectionOpen 数据库打开连接数
	DBConnectionOpen = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "kubeai_model_manager_db_connections_open",
		Help: "Number of open database connections",
	})

	// DBConnectionInUse 数据库使用中连接数
	DBConnectionInUse = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "kubeai_model_manager_db_connections_in_use",
		Help: "Number of in-use database connections",
	})

	// DBConnectionIdle 数据库空闲连接数
	DBConnectionIdle = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "kubeai_model_manager_db_connections_idle",
		Help: "Number of idle database connections",
	})
)

// Handler 返回 Prometheus metrics HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}
