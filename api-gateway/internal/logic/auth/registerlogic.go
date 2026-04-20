// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package auth

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"kubeai-api-gateway/internal/model"

	"kubeai-api-gateway/internal/svc"
	"kubeai-api-gateway/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogic) Register(req *types.RegisterRequest) (resp *types.RegisterResponse, err error) {
	var exists model.User
	err = l.svcCtx.DB.Where("username = ?", req.Username).First(&exists).Error
	if err == nil {
		return nil, errors.New("username already exists")
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	user := &model.User{
		Username: req.Username,
		Password: req.Password,
		Role:     "user",
	}
	err = l.svcCtx.DB.Create(user).Error
	if err != nil {
		return nil, err
	}
	return &types.RegisterResponse{
		Message: "register success",
		UserID:  user.ID,
	}, nil
}
