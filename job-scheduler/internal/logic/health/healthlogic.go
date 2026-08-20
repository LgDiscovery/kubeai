// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package health

import (
	"context"
	"time"

	"kubeai-job-scheduler/internal/svc"
	"kubeai-job-scheduler/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type HealthLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewHealthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HealthLogic {
	return &HealthLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Health 健康检查（liveness）
func (l *HealthLogic) Health() (resp *types.CommonResp, err error) {
	resp = &types.CommonResp{
		Code:    0,
		Message: "success",
		Data: map[string]interface{}{
			"status":    "UP",
			"service":   "job-scheduler",
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}
	return
}

// CheckDependencies 检查所有依赖是否健康
func (l *HealthLogic) CheckDependencies() map[string]interface{} {
	result := make(map[string]interface{})

	// 检查数据库连接
	dbStatus := "UP"
	dbMsg := ""
	ctx, cancel := context.WithTimeout(l.ctx, 5*time.Second)
	defer cancel()

	sqlDB, err := l.svcCtx.DB.DB()
	if err != nil {
		dbStatus = "DOWN"
		dbMsg = err.Error()
	} else {
		if err := sqlDB.PingContext(ctx); err != nil {
			dbStatus = "DOWN"
			dbMsg = err.Error()
		} else {
			stats := sqlDB.Stats()
			result["db_connections"] = map[string]interface{}{
				"open":   stats.OpenConnections,
				"in_use": stats.InUse,
				"idle":   stats.Idle,
			}
		}
	}
	result["database"] = map[string]interface{}{
		"status":  dbStatus,
		"message": dbMsg,
	}

	// 检查 Redis 连接
	redisStatus := "UP"
	redisMsg := ""
	redisCtx, redisCancel := context.WithTimeout(l.ctx, 5*time.Second)
	defer redisCancel()
	if err := l.svcCtx.RedisClient.Ping(redisCtx).Err(); err != nil {
		redisStatus = "DOWN"
		redisMsg = err.Error()
	}
	result["redis"] = map[string]interface{}{
		"status":  redisStatus,
		"message": redisMsg,
		"type":    "cluster",
	}

	// 检查 K8s API 连接
	k8sStatus := "UP"
	k8sMsg := ""
	k8sCtx, k8sCancel := context.WithTimeout(l.ctx, 5*time.Second)
	defer k8sCancel()
	_, err = l.svcCtx.K8sClient.ServerVersion()
	if err != nil {
		k8sStatus = "DOWN"
		k8sMsg = err.Error()
	}
	_ = k8sCtx
	result["kubernetes"] = map[string]interface{}{
		"status":    k8sStatus,
		"message":   k8sMsg,
		"namespace": l.svcCtx.Config.K8s.Namespace,
	}

	// 检查 ModelManager 服务连接
	modelMgrStatus := "UP"
	modelMgrMsg := ""
	if l.svcCtx.ModelManagerClient != nil {
		// 这里可以调用 ModelManager 的健康检查接口
		// 暂时只检查客户端是否初始化
		modelMgrStatus = "UP"
	} else {
		modelMgrStatus = "DOWN"
		modelMgrMsg = "model manager client not initialized"
	}
	result["model_manager"] = map[string]interface{}{
		"status":  modelMgrStatus,
		"message": modelMgrMsg,
		"url":     l.svcCtx.Config.ModelManager.URL,
	}

	return result
}
