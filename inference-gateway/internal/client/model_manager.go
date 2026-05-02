package client

import (
	"bytes"
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

type Model struct {
	ID          uint           `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,optional"`
	Framework   string         `json:"framework,optional"`
	TaskType    string         `json:"task_type,optional"`
	Owner       string         `json:"owner,optional"`
	Labels      string         `json:"labels,optional"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
	Versions    []ModelVersion `json:"versions,optional"`
}

// ModelRegisterRequest 模型注册请求结构体
type ModelRegisterRequest struct {
	ModelName       string            `json:"model_name"`
	StoragePath     string            `json:"storage_path"`
	Framework       string            `json:"framework"`
	Version         string            `json:"version"`
	TaskType        string            `json:"task_type,omitempty"`
	Description     string            `json:"description,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	TrainingJobName string            `json:"training_job_name,omitempty"`
	Namespace       string            `json:"namespace,omitempty"`
	ModelID         string            `json:"model_id,omitempty"`
}

// ModelRegisterResponse 对齐 model-manager 接口的响应结构体
type ModelRegisterResponse struct {
	ModelID string `json:"model_id"`
	Success bool   `json:"success"`
	Message string `json:"message"`
	Version string `json:"version,omitempty"`
}

func (c *ModelManagerClient) GetModel(ctx context.Context, modelName string) (*Model, error) {
	url := fmt.Sprintf("%s/api/v1/models/%s", c.baseURL, modelName)
	httpResp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	model := &Model{}
	if err := json.NewDecoder(httpResp.Body).Decode(model); err != nil {
		return nil, err
	}
	return model, nil
}

// CheckModelAvailable 校验模型是否可用，返回模型版本信息
func (c *ModelManagerClient) CheckModelAvailable(ctx context.Context, modelName, version string) (*ModelVersion, error) {
	url := fmt.Sprintf("%s/api/v1/models/%s/versions/%s", c.baseURL, modelName, version)
	resp, err := c.httpClient.Get(url)
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
	resp, err := c.httpClient.Get(url)
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
	Framework    string `json:"framework"`
	// 模型在 MinIO/S3 中的路径，例如: s3://models/bert/v1/model.tar.gz
	StoragePath string `json:"storagePath"`
}

// GetModelMetadata GetModel fetches the model metadata
func (c *ModelManagerClient) GetModelMetadata(ctx context.Context, modelName, modelVersion string) (*ModelMetadata, error) {
	url := fmt.Sprintf("%s/api/v1/models/%s/versions/%s/metadata", c.baseURL, modelName, modelVersion)
	resp, err := c.httpClient.Get(url)
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

func (c *ModelManagerClient) RegisterModel(ctx context.Context, registerRequest *ModelRegisterRequest) (*ModelRegisterResponse, error) {
	url := fmt.Sprintf("%s/api/v1/models/register", c.baseURL)
	registerRequestJSON, err := json.Marshal(registerRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal register request: %w", err)
	}
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(registerRequestJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to call model manager: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to call model manager: invalid status code: %d", resp.StatusCode)
	}
	var modelRegisterResponse ModelRegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelRegisterResponse); err != nil {
		return nil, fmt.Errorf("failed to decode model register response: %w", err)
	}
	return &modelRegisterResponse, nil
}
