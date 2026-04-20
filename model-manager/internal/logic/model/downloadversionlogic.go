// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package model

import (
	"context"
	"io"

	"github.com/zeromicro/go-zero/core/logx"
	"kubeai-model-manager/internal/svc"
)

type DownloadVersionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDownloadVersionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DownloadVersionLogic {
	return &DownloadVersionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DownloadVersionLogic) DownloadVersion(name, version string) (io.ReadCloser, string, error) {
	m, err := l.svcCtx.ModelRepo.GetByName(l.ctx, name)
	if err != nil {
		l.Logger.Errorf("DownloadVersion err: %v", err)
		return nil, "model not found", err
	}
	v, err := l.svcCtx.VersionRepo.GetByModelAndVersion(l.ctx, m.ID, version)
	if err != nil {
		l.Logger.Errorf("DownloadVersion err: %v", err)
		return nil, "version not found", err
	}
	reader, err := l.svcCtx.MinIOClient.Download(l.ctx, v.StoragePath)
	if err != nil {
		l.Logger.Errorf("DownloadVersion err: %v", err)
		return nil, "download version failed", err
	}
	return reader, v.StoragePath, nil
}
