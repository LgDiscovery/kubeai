// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package middleware

import (
	"github.com/zeromicro/go-zero/core/limit"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest/httpx"
	"kubeai-api-gateway/internal/config"
	"net/http"
)

type RateLimitMiddleware struct {
	c           config.RateLimit
	redisClient *redis.Redis
}

func NewRateLimitMiddleware(config config.RateLimit, redisClient *redis.Redis) *RateLimitMiddleware {
	return &RateLimitMiddleware{c: config, redisClient: redisClient}
}

func (m *RateLimitMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. 未开启限流：返回透传中间件
		if !m.c.Enabled {
			next(w, r) // 直接透传，不做任何处理
			return
		}

		// 2. 初始化限流器（分布式限流）
		limiter := limit.NewPeriodLimit(
			int(m.c.Interval), // period：时间窗口（秒）
			m.c.Rate,          // quota：时间窗口内最大请求数
			m.redisClient,     // limitStore：Redis客户端
			m.c.KeyPrefix,     // keyPrefix：Redis key前缀
		)

		// 4. 返回中间件
		// 以客户端IP作为限流key
		key := r.RemoteAddr
		code, err := limiter.Take(key)
		if err != nil {
			logx.Errorf("rate limit error: %v, ip: %s", err, key)
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]string{
				"error": "rate limit internal error",
			})
			return
		}

		// 超过限流：返回429
		if code == limit.OverQuota {
			logx.Errorf("rate limit exceeded: ip=%s, path=%s", key, r.URL.Path)
			httpx.WriteJson(w, http.StatusTooManyRequests, map[string]string{
				"error": "too many requests",
			})
			return
		}

		// 限流通过：调用next HandlerFunc
		next(w, r)
	}
}
