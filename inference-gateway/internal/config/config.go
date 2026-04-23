package config

import (
	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/rest"
	"time"
)

type Config struct {
	rest.RestConf
	Etcd         discov.EtcdConf `json:",optional"`
	Redis        RedisConfig
	Database     DatabaseConfig
	ModelManager ModelManagerConfig
	K8s          K8sConfig
	Metrics      MetricsConfig
	Scheduler    SchedulerConfig
	ResourceSync ResourceSyncConfig
	Log          LogConfig
}

type ResourceSyncConfig struct {
	Interval time.Duration
}

type RedisConfig struct {
	Addr          string
	Password      string
	DB            int
	Streams       StreamsConfig
	ConsumerGroup string
}

type DatabaseConfig struct {
	Host         string
	Port         int
	User         string
	Password     string
	DBName       string
	SSLMode      string
	MaxIdleConns int `json:",default=10"`
	MaxOpenConns int `json:",default=100"`
}

type StreamsConfig struct {
	Inference  string
	Training   string
	DeadLetter string
}

type ModelManagerConfig struct {
	URL     string
	Timeout time.Duration
}

type K8sConfig struct {
	Namespace string
}

type MetricsConfig struct {
	Enabled bool
	Path    string
}

type SchedulerConfig struct {
	Algorithm        string
	EnableGPUPacking bool
	GPUBinpackWeight float64
	CacheLabelPrefix string `json:",optional"` // GPU 亲和性缓存标签前缀
}

type LogConfig struct {
	ServiceName string `json:"serviceName"`
	Mode        string `json:"mode"`
	Encoding    string `json:"encoding"`
	Level       string `json:",default=info"`
	TimeFormat  string `json:"timeFormat"`
}
