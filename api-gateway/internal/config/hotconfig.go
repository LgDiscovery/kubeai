package config

import (
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// HotConfig 可通过 etcd 热更新的配置项
type HotConfig struct {
	Database  DatabaseConfig  `mapstructure:"database" json:"database"`
	RedisConf redis.RedisConf `mapstructure:"redisConf" json:"redisConf"`
	RateLimit RateLimit       `mapstructure:"rateLimit" json:"rateLimit"`
	Auth      Auth            `mapstructure:"auth" json:"auth"`
	Upstreams Upstreams       `mapstructure:"upstreams" json:"upstreams"`
	Metrics   MetricsConfig   `mapstructure:"metrics" json:"metrics"`
	Log       LogConfig       `mapstructure:"log" json:"log"`
}

// OnChange 实现 go-zero 配置监听回调
func (h *HotConfig) OnChange() {
	// 更新日志级别
	if err := logx.SetUp(logx.LogConf{Level: h.Log.Level}); err != nil {
		logx.Errorf("update log level failed: %v", err)
	}
	// 其他运行时配置（如限流桶重建）将在服务上下文中处理
}
