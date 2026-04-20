// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"time"
)

type Config struct {
	rest.RestConf
	Etcd      EtcdConf        `json:",optional"`
	Database  DatabaseConfig  `mapstructure:"database" json:"database"`
	RedisConf redis.RedisConf `mapstructure:"redisConf" json:"redisConf"`
	RateLimit RateLimit       `mapstructure:"rateLimit" json:"rateLimit"`
	Auth      Auth            `mapstructure:"auth" json:"auth"`
	Upstreams Upstreams       `mapstructure:"upstreams" json:"upstreams"`
	Metrics   MetricsConfig   `mapstructure:"metrics" json:"metrics"`
	Log       LogConfig       `mapstructure:"log" json:"log"`
	Discovery DiscoveryConfig `mapstructure:"discovery" json:"discovery"`
	HotConfig HotConfig       `mapstructure:"hotConfig" json:"hotConfig"`
}

type Auth struct {
	AccessSecret string `mapstructure:"accessSecret" json:"accessSecret"`
	AccessExpire int64  `mapstructure:"accessExpire" json:"accessExpire"`
}

type Upstreams struct {
	ModelManager         Upstream `mapstructure:"modelManager" json:"modelManager"`
	JobScheduler         Upstream `mapstructure:"jobScheduler" json:"jobScheduler"`
	InferenceGateway     Upstream `mapstructure:"inferenceGateway" json:"inferenceGateway"`
	ObservabilityGateway Upstream `mapstructure:"observabilityGateway" json:"observabilityGateway"`
}

type Upstream struct {
	URL string `mapstructure:"url" json:"url"`
}

type RateLimit struct {
	Enabled   bool   `mapstructure:"enabled" json:"enabled" json:",default=true"`                          // 是否开启限流，默认开启
	Rate      int    `mapstructure:"rate" json:"rate" json:",default=100"`                                 // 时间窗口内最大请求数
	Interval  int    `mapstructure:"interval" json:"interval" json:",default=60s"`                         // 时间窗口，支持 60s/1m/1h 等格式
	KeyPrefix string `mapstructure:"keyPrefix" json:"keyPrefix" json:",default=kubeai:gateway:ratelimit:"` // Redis key前缀
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

type MetricsConfig struct {
	Enabled bool   `json:",default=true"`
	Path    string `json:",default=/api/v1/metrics"`
}

type LogConfig struct {
	ServiceName string `mapstructure:"serviceName" json:"serviceName"`
	Mode        string `mapstructure:"mode" json:"mode"`
	Encoding    string `mapstructure:"encoding" json:"encoding"`
	Level       string `json:",default=info"`
}

type DiscoveryConfig struct {
	ModelManagerKey         string `mapstructure:"modelManagerKey" json:"modelManagerKey"`
	TaskSchedulerKey        string `mapstructure:"taskSchedulerKey" json:"taskSchedulerKey"`
	InferenceGatewayKey     string `mapstructure:"inferenceGatewayKey" json:"inferenceGatewayKey"`
	ObservabilityGatewayKey string `mapstructure:"observabilityGatewayKey" json:"observabilityGatewayKey"`
}

type EtcdConf struct {
	Hosts              []string
	Key                string
	User               string        `json:",optional"`
	Pass               string        `json:",optional"`
	CertFile           string        `json:",optional"`
	CertKeyFile        string        `json:",optional"`
	CACertFile         string        `json:",optional"`
	InsecureSkipVerify bool          `json:",optional"`
	DialTimeout        time.Duration `json:",optional"`
}
