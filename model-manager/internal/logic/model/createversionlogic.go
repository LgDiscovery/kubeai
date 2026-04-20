// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package model

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"kubeai-model-manager/internal/model"
	"kubeai-model-manager/pkg/metrics"
	"mime/multipart"
	"time"

	"kubeai-model-manager/internal/svc"
	"kubeai-model-manager/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateVersionLogic struct {
	logx.Logger
	ctx        context.Context
	svcCtx     *svc.ServiceContext
	fileHeader *multipart.FileHeader
}

func NewCreateVersionLogic(ctx context.Context, svcCtx *svc.ServiceContext, fileHeader *multipart.FileHeader) *CreateVersionLogic {
	return &CreateVersionLogic{
		Logger:     logx.WithContext(ctx),
		ctx:        ctx,
		svcCtx:     svcCtx,
		fileHeader: fileHeader,
	}
}

func (l *CreateVersionLogic) CreateVersion(req *types.CreateVersionReq) (resp *types.ModelVersion, err error) {
	m, err := l.svcCtx.ModelRepo.GetByName(l.ctx, req.Name)
	if err != nil {
		l.Logger.Errorf("CreateVersion err:%v", err)
		return nil, errors.New(fmt.Sprintf("model-manager get model fail, %s", err.Error()))
	}
	// 检查模型版本是否存在
	if _, err := l.svcCtx.VersionRepo.GetByModelAndVersion(l.ctx, m.ID, req.Version); err == nil {
		l.Logger.Errorf("CreateVersion err:%v", err)
		return nil, errors.New("model version already exists")
	}

	file, err := l.fileHeader.Open()
	if err != nil {
		l.Logger.Errorf("CreateVersion err:%v", err)
		return nil, errors.New(fmt.Sprintf("model-manager open file fail, %s", err.Error()))
	}
	defer file.Close()
	size := l.fileHeader.Size
	hash := sha256.New()
	teeReader := io.TeeReader(file, hash)

	// 上传到 MinIO
	storagePath, err := l.svcCtx.MinIOClient.Upload(l.ctx, req.Name, req.Version, teeReader, size)
	if err != nil {
		l.Logger.Errorf("CreateVersion err:%v", err)
		return nil, errors.New(fmt.Sprintf("model-manager upload file fail, %s", err.Error()))
	}
	// 计算 checksum
	checksum := hex.EncodeToString(hash.Sum(nil))

	version := &model.ModelVersion{
		ModelID:      m.ID,
		Version:      req.Version,
		Description:  req.Description,
		StoragePath:  storagePath,
		Framework:    req.Framework,
		FrameworkVer: req.FrameworkVersion,
		Metrics:      req.Metrics,
		Parameters:   req.Parameters,
		Size:         size,
		Checksum:     checksum,
		Status:       "active",
	}
	// 保存到数据库
	if err := l.svcCtx.VersionRepo.Create(l.ctx, version); err != nil {
		l.Logger.Errorf("CreateVersion error: %v", err)
		// 上传到 MinIO 失败，删除文件
		l.svcCtx.MinIOClient.Delete(l.ctx, storagePath)
		return nil, err
	}
	metrics.ModelVersionTotal.WithLabelValues("active").Inc()
	metrics.ModelHealthStatus.WithLabelValues(req.Name, req.Version).Set(1)

	return toVersionResponse(version), nil
}

func toVersionResponse(v *model.ModelVersion) *types.ModelVersion {
	return &types.ModelVersion{
		ID:           v.ID,
		ModelID:      v.ModelID,
		Version:      v.Version,
		Description:  v.Description,
		StoragePath:  v.StoragePath,
		Framework:    v.Framework,
		FrameworkVer: v.FrameworkVer,
		Metrics:      v.Metrics,
		Parameters:   v.Parameters,
		Size:         v.Size,
		Checksum:     v.Checksum,
		Status:       v.Status,
		CreatedAt:    v.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    v.UpdatedAt.Format(time.RFC3339),
	}
}
