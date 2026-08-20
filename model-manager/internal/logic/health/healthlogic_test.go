package health

import (
	"context"
	"testing"

	"kubeai-model-manager/internal/svc"
	"kubeai-model-manager/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockServiceContext 模拟 ServiceContext
type MockServiceContext struct {
	mock.Mock
	svc.ServiceContext
}

func TestHealthLogic_Health(t *testing.T) {
	// 准备
	ctx := context.Background()
	svcCtx := &svc.ServiceContext{}
	logic := NewHealthLogic(ctx, svcCtx)

	// 执行
	resp, err := logic.Health()

	// 断言
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "success", resp.Message)
	assert.NotNil(t, resp.Data)

	data, ok := resp.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "UP", data["status"])
	assert.Equal(t, "model-manager", data["service"])
	assert.NotEmpty(t, data["timestamp"])
}

func TestHealthLogic_CheckDependencies(t *testing.T) {
	// 准备
	ctx := context.Background()
	svcCtx := &svc.ServiceContext{}
	logic := NewHealthLogic(ctx, svcCtx)

	// 执行
	result := logic.CheckDependencies()

	// 断言
	assert.NotNil(t, result)
	assert.Contains(t, result, "database")
	assert.Contains(t, result, "minio")

	// 检查 database 部分
	dbInfo, ok := result["database"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, dbInfo, "status")
	assert.Contains(t, dbInfo, "message")

	// 检查 minio 部分
	minioInfo, ok := result["minio"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, minioInfo, "status")
	assert.Contains(t, minioInfo, "message")
	assert.Contains(t, minioInfo, "bucket")
}

func TestReadyLogic_Ready(t *testing.T) {
	// 准备
	ctx := context.Background()
	svcCtx := &svc.ServiceContext{}
	logic := NewReadyLogic(ctx, svcCtx)

	// 执行
	resp, err := logic.Ready()

	// 断言
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 0, resp.Code)
	assert.NotNil(t, resp.Data)

	data, ok := resp.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, data, "status")
	assert.Contains(t, data, "service")
	assert.Contains(t, data, "timestamp")
	assert.Contains(t, data, "dependencies")
}

func TestCommonResp_Structure(t *testing.T) {
	// 测试 CommonResp 结构
	resp := &types.CommonResp{
		Code:    0,
		Message: "success",
		Data:    map[string]interface{}{"key": "value"},
	}

	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "success", resp.Message)
	assert.NotNil(t, resp.Data)
}
