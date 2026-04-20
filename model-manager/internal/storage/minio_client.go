package storage

import (
	"context"
	"fmt"
	"io"
	"kubeai-model-manager/internal/config"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOClient struct {
	client *minio.Client
	bucket string
}

func NewMinIOClient(minioConfig config.MinIOConfig) (*MinIOClient, error) {
	client, err := minio.New(minioConfig.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioConfig.AccessKey, minioConfig.SecretKey, ""),
		Secure: minioConfig.UseSSL,
	})
	if err != nil {
		return nil, err
	}
	// 确保 bucket 存在
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, minioConfig.Bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err = client.MakeBucket(ctx, minioConfig.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}
	return &MinIOClient{client: client, bucket: minioConfig.Bucket}, nil
}

// Upload 上传模型文件，返回存储路径和文件大小
func (m *MinIOClient) Upload(ctx context.Context, modelName, version string, reader io.Reader, size int64) (storagePath string, err error) {
	// 生成唯一文件名，路径格式：models/{modelName}/{version}/{uuid}.bin
	ext := ".bin"
	objectName := fmt.Sprintf("%s/%s/%s%s", modelName, version, uuid.New().String(), ext)

	_, err = m.client.PutObject(ctx, m.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return "", err
	}
	return objectName, nil
}

// Download 下载模型文件
func (m *MinIOClient) Download(ctx context.Context, objectName string) (io.ReadCloser, error) {
	return m.client.GetObject(ctx, m.bucket, objectName, minio.GetObjectOptions{})
}

// Delete 删除模型文件
func (m *MinIOClient) Delete(ctx context.Context, objectName string) error {
	return m.client.RemoveObject(ctx, m.bucket, objectName, minio.RemoveObjectOptions{})
}

// GetPresignedURL 生成临时下载链接 (有效期1小时)
func (m *MinIOClient) GetPresignedURL(ctx context.Context, objectName string) (string, error) {
	reqParams := make(url.Values)
	presignedURL, err := m.client.PresignedGetObject(ctx, m.bucket, objectName, time.Hour, reqParams)
	if err != nil {
		return "", err
	}
	return presignedURL.String(), nil
}
