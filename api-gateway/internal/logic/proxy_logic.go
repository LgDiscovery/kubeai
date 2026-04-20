package logic

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
	"kubeai-api-gateway/internal/svc"
)

type ProxyLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProxyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ProxyLogic {
	return &ProxyLogic{ctx: ctx, svcCtx: svcCtx}
}

// ProxyRequest 根据路径选择上游服务并执行反向代理
func (l *ProxyLogic) ProxyRequest(w http.ResponseWriter, r *http.Request) error {
	path := r.URL.Path
	var targetHost string
	var err error
	switch {
	case strings.HasPrefix(path, "/api/v1/models"):
		targetHost, err = l.svcCtx.Discovery.GetModelManagerEndpoint()
	case strings.HasPrefix(path, "/api/v1/jobs"):
		targetHost, err = l.svcCtx.Discovery.GetTaskSchedulerEndpoint()
	case strings.HasPrefix(path, "/api/v1/inference"):
		targetHost, err = l.svcCtx.Discovery.GetInferenceGatewayEndpoint()
	case strings.HasPrefix(path, "/api/v1/observability"):
		targetHost, err = l.svcCtx.Discovery.GetObservabilityGatewayEndpoint()
	default:
		return fmt.Errorf("no upstream found for path: %s", path)
	}
	if err != nil {
		logx.Errorf("discovery get endpoint failed: %v", err)
		return err
	}
	targetURL := fmt.Sprintf("http://%s", targetHost)
	target, err := url.Parse(targetURL)
	if err != nil {
		logx.Errorf("parse upstream url failed: %v", err)
		return err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	// 自定义 Director 以修改请求头
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// 透传用户信息（从上下文中提取）
		if userID, ok := l.ctx.Value("user_id").(uint); ok {
			req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
		}
		if username, ok := l.ctx.Value("username").(string); ok {
			req.Header.Set("X-Username", username)
		}
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Origin-Host", target.Host)
	}
	// 错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logx.Errorf("proxy error: %v", err)
		http.Error(w, "Service Unavailable", http.StatusBadGateway)
	}

	proxy.ServeHTTP(w, r)
	return nil
}
