package handler

import (
	"github.com/zeromicro/go-zero/rest"
	"kubeai-api-gateway/internal/handler/business"
	"kubeai-api-gateway/internal/svc"
	"net/http"
)

// RegisterCommHandlers 为每个服务路径注册通配路由
func RegisterCommHandlers(server *rest.Server, ctx *svc.ServiceContext) {
	allMethods := []string{
		http.MethodGet, http.MethodPost, http.MethodPut,
		http.MethodDelete, http.MethodPatch, http.MethodHead,
		http.MethodOptions,
	}

	paths := map[string]http.HandlerFunc{
		"/inference/*": business.ProxyInferenceHandler(ctx),
		"/jobs/*":      business.ProxyJobHandler(ctx),
		"/models/*":    business.ProxyModelHandler(ctx),
		"/observer/*":  business.ProxyObserverHandler(ctx),
	}

	for pattern, handler := range paths {
		for _, method := range allMethods {
			server.AddRoutes(
				rest.WithMiddlewares(
					[]rest.Middleware{ctx.CorsMiddleware, ctx.AuthMiddleware, ctx.RateLimitMiddleware, ctx.MetricsMiddleware},
					[]rest.Route{
						{
							Method:  method,
							Path:    pattern,
							Handler: handler,
						},
					}...,
				),
				rest.WithPrefix("/api/v1"),
			)
		}
	}
}
