package config

import (
	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	Database DatabaseConfig
	MinIO    MinIOConfig
	Metrics  MetricsConfig
	Etcd     discov.EtcdConf `json:",optional"`
	Log      LogConfig       `mapstructure:"log" json:"log"`
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

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool `json:",default=false"`
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
