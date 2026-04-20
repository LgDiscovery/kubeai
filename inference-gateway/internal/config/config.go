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
	ModelManager ModelManagerConfig
	K8s          K8sConfig
	Metrics      MetricsConfig
}

type RedisConfig struct {
	Addr          string
	Password      string
	DB            int
	Streams       StreamsConfig
	ConsumerGroup string
}

type StreamsConfig struct {
	Inference string
	Training  string
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
