// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

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
	ModelManager ModelManagerConfing
	K8s          K8sConfing
	ResourceSync ResourceSyncConfing
	Log          LogConfig
	Metrics      MetricsConfing
	Database     DatabaseConfig
}

type RedisConfig struct {
	Addr          string
	Password      string
	DB            int
	Streams       StreamsConfig
	ConsumerGroup string
	MaxRetries    int
	Expiration    time.Duration
	RetryBackoff  int //ms
}

type StreamsConfig struct {
	Inference string
	Training  string
}

type ModelManagerConfing struct {
	URL     string
	Timeout time.Duration
}

type K8sConfing struct {
	Namespace         string
	InferenceJobImage string
	TrainingJobImage  string
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

type ResourceSyncConfing struct {
	Interval time.Duration
}

type LogConfig struct {
	ServiceName string `json:"serviceName"`
	Mode        string `json:"mode"`
	Encoding    string `json:"encoding"`
	Level       string `json:",default=info"`
	TimeFormat  string `json:"timeFormat"`
}

type MetricsConfing struct {
	Enabled  bool
	Interval time.Duration
}
