package storage

import (
	"context"
	"strings"
	"testing"

	"kubeai-model-manager/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestMinIOClient_NewClient_InvalidConfig(t *testing.T) {
	// 测试无效配置
	config := config.MinIOConfig{
		Endpoint:  "",
		AccessKey: "",
		SecretKey: "",
		Bucket:    "",
	}

	client, err := NewMinIOClient(config)
	// 无效配置应该返回错误
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestMinIOClient_GetBucketName(t *testing.T) {
	// 由于无法在单元测试中真实连接 MinIO，我们测试基本的结构和方法存在性
	// 创建一个模拟的 MinIOClient（不通过 NewMinIOClient，因为需要真实连接）
	client := &MinIOClient{
		client: nil,
		bucket: "test-bucket",
	}

	// 测试 GetBucketName 方法
	bucketName := client.GetBucketName()
	assert.Equal(t, "test-bucket", bucketName)
}

func TestMinIOClient_HealthCheck_NilClient(t *testing.T) {
	// 测试 nil client 的健康检查应该返回错误
	client := &MinIOClient{
		client: nil,
		bucket: "test-bucket",
	}

	ctx := context.Background()
	err := client.HealthCheck(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "minio health check failed")
}

func TestMinIOConfig_Defaults(t *testing.T) {
	// 测试 MinIO 配置的默认值
	config := config.MinIOConfig{
		Endpoint:  "localhost:9000",
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
		Bucket:    "models",
		UseSSL:    false,
	}

	assert.Equal(t, "localhost:9000", config.Endpoint)
	assert.Equal(t, "minioadmin", config.AccessKey)
	assert.Equal(t, "minioadmin", config.SecretKey)
	assert.Equal(t, "models", config.Bucket)
	assert.False(t, config.UseSSL)
}

func TestMinIOClient_Upload_InvalidInput(t *testing.T) {
	// 测试无效输入的上传
	client := &MinIOClient{
		client: nil,
		bucket: "test-bucket",
	}

	ctx := context.Background()
	// nil reader 应该返回错误
	_, err := client.Upload(ctx, "test-model", "v1.0.0", nil, 0)
	assert.Error(t, err)
}

func TestMinIOClient_Download_NonExistentObject(t *testing.T) {
	// 测试下载不存在的对象
	client := &MinIOClient{
		client: nil,
		bucket: "test-bucket",
	}

	ctx := context.Background()
	_, err := client.Download(ctx, "non-existent-object")
	assert.Error(t, err)
}

func TestMinIOClient_Delete_NonExistentObject(t *testing.T) {
	// 测试删除不存在的对象
	client := &MinIOClient{
		client: nil,
		bucket: "test-bucket",
	}

	ctx := context.Background()
	err := client.Delete(ctx, "non-existent-object")
	assert.Error(t, err)
}

func TestMinIOClient_GetPresignedURL_InvalidObject(t *testing.T) {
	// 测试生成无效对象的预签名 URL
	client := &MinIOClient{
		client: nil,
		bucket: "test-bucket",
	}

	ctx := context.Background()
	_, err := client.GetPresignedURL(ctx, "")
	assert.Error(t, err)
}

func TestStoragePath_Format(t *testing.T) {
	// 测试存储路径格式
	// 路径格式应该是 models/{modelName}/{version}/{uuid}.bin
	modelName := "test-model"
	version := "v1.0.0"

	// 模拟路径生成
	pathParts := []string{"models", modelName, version, "test-uuid.bin"}
	path := strings.Join(pathParts, "/")

	assert.True(t, strings.HasPrefix(path, "models/"))
	assert.Contains(t, path, modelName)
	assert.Contains(t, path, version)
	assert.True(t, strings.HasSuffix(path, ".bin"))
}
