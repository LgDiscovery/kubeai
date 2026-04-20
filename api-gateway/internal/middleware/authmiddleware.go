// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package middleware

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"kubeai-api-gateway/pkg/jwt"
	"net/http"
	"strings"
)

// 定义自定义 Context Key 类型
type ctxKey string

const (
	CtxUserID   ctxKey = "user_id"
	CtxRole     ctxKey = "role"
	CtxUsername ctxKey = "username"
)

type AuthMiddleware struct {
}

func NewAuthMiddleware() *AuthMiddleware {
	return &AuthMiddleware{}
}

func (m *AuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid Authorization header"})
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := jwt.ParseToken(token)
		if err != nil {
			logx.Errorf("parse token failed: %v", err)
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}
		// 将用户信息存入请求上下文（go-zero 风格）
		ctx := r.Context()
		ctx = context.WithValue(ctx, CtxUserID, claims.UserID)
		ctx = context.WithValue(ctx, CtxUsername, claims.Username)
		ctx = context.WithValue(ctx, CtxRole, claims.Role)
		next(w, r.WithContext(ctx))
	}
}
