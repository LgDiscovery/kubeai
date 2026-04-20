// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package auth

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"kubeai-api-gateway/internal/model"
	"kubeai-api-gateway/pkg/jwt"

	"kubeai-api-gateway/internal/svc"
	"kubeai-api-gateway/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginRequest) (resp *types.LoginResponse, err error) {
	var user model.User
	err = l.svcCtx.DB.Where("username = ? AND password = ?", req.Username, req.Password).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid username or password")
		}
		return nil, err
	}
	token, err := jwt.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, err
	}
	return &types.LoginResponse{
		Token:    token,
		UserID:   user.ID,
		Role:     user.Role,
		Username: user.Username,
	}, nil
}
