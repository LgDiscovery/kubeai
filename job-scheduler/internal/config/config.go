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
	Etcd             discov.EtcdConf `json:",optional"`
	Redis            RedisConfig
	ModelManager     ModelManagerConfing
	InferenceGateway InferenceGatewayConfing
	K8s              K8sConfing
	Scheduler        SchedulerConfig
	ResourceSync     ResourceSyncConfing
	Log              LogConfing
	Metrics          MetricsConfing
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

type InferenceGatewayConfing struct {
	URL     string
	Timeout time.Duration
}

type K8sConfing struct {
	Namespace         string
	InferenceJobImage string
	TrainingJobImage  string
}

type ResourceSyncConfing struct {
	Interval time.Duration
}

type LogConfing struct {
	ServiceName string
	Mode        string
	Level       string
	Encoding    string
}

type MetricsConfing struct {
	Enabled  bool
	Interval time.Duration
}

type SchedulerConfig struct {
	Algorithm        string
	EnableGPUPacking bool
	GPUBinpackWeight float64
}
