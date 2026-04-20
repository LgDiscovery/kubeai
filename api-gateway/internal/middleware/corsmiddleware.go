// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package middleware

import "net/http"

type CorsMiddleware struct {
}

func NewCorsMiddleware() *CorsMiddleware {
	return &CorsMiddleware{}
}

func (m *CorsMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		method := r.Method
		// 生产环境建议替换为具体域名，如 "https://your-frontend.com"
		allowOrigin := r.Header.Get("Origin")
		if allowOrigin == "" {
			allowOrigin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-type,AccessToken,X-CSRF-Token,Authorization,Token,X-Token,X-User")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length,Access-Control-Allow-Origin,Access-Control-Allow-Headers,Content-Type")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400") // 预检请求缓存时间，减少OPTIONS请求

		if method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}
