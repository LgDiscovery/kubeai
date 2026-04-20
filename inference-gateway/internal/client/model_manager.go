package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ModelManagerClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewModelManagerClient(baseURL string, timeout time.Duration) *ModelManagerClient {
	return &ModelManagerClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type ModelVersion struct {
	ID          uint   `json:"id"`
	ModelID     uint   `json:"model_id"`
	Version     string `json:"version"`
	StoragePath string `json:"storage_path"`
	Framework   string `json:"framework"`
	Status      string `json:"status"`
	Size        int64  `json:"size"`
	Checksum    string `json:"checksum"`
}

// CheckModelAvailable 校验模型是否可用，返回模型版本信息
func (c *ModelManagerClient) CheckModelAvailable(ctx context.Context, modelName, version string) (*ModelVersion, error) {
	url := fmt.Sprintf("%s/api/v1/models/%s/versions/%s", c.baseURL, modelName, version)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("model version not found or unavailable, status: %d", resp.StatusCode)
	}

	var result struct {
		Code int          `json:"code"`
		Data ModelVersion `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Data.Status != "active" {
		return nil, fmt.Errorf("model version is not active, status: %s", result.Data.Status)
	}
	return &result.Data, nil
}

// GetModelDownloadURL 获取模型下载预签名 URL（用于传递给推理 Pod）
func (c *ModelManagerClient) GetModelDownloadURL(ctx context.Context, modelName, version string) (string, error) {
	url := fmt.Sprintf("%s/api/v1/models/%s/versions/%s/download?presigned=true", c.baseURL, modelName, version)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Code int `json:"code"`
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Data.URL, nil
}

// ModelMetadata represents the response from the Model Manager API.
type ModelMetadata struct {
	ModelName    string `json:"modelName"`
	ModelVersion string `json:"modelVersion"`
	// 模型在 MinIO/S3 中的路径，例如: s3://models/bert/v1/model.tar.gz
	StoragePath string `json:"storagePath"`
}

// GetModelMetadata GetModel fetches the model metadata
func (c *ModelManagerClient) GetModelMetadata(modelName, modelVersion string) (*ModelMetadata, error) {
	url := fmt.Sprintf("%s/api/v1/models/%s/versions/%s/metadata", c.baseURL, modelName, modelVersion)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call model manager: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to call model manager: invalid status code: %d", resp.StatusCode)
	}

	var modelMetadata ModelMetadata
	if err := json.NewDecoder(resp.Body).Decode(&modelMetadata); err != nil {
		return nil, fmt.Errorf("failed to decode model metadata: %w", err)
	}
	return &modelMetadata, nil
}
