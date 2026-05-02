package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"kubeai-inference-gateway/internal/types"
	"net/http"
	"time"
)

type JobScheduleClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewJobScheduleClient(baseURL string, timeout time.Duration) *JobScheduleClient {
	return &JobScheduleClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// CallBackTaskStatusReq 回调请求结构体
type CallBackTaskStatusReq struct {
	TaskID     string `json:"task_id"`
	ModelName  string `json:"model_name"`
	Phase      string `json:"phase"`
	Reason     string `json:"reason,omitempty"`
	Message    string `json:"message,omitempty"`
	ModelID    string `json:"model_id,omitempty"`
	NodeName   string `json:"node_name,omitempty"`
	FinishedAt string `json:"finished_at,omitempty"`
}

func (j *JobScheduleClient) CallBackTaskStatus(ctx context.Context, req *CallBackTaskStatusReq) (*types.CommonResp, error) {
	url := fmt.Sprintf("%s/api/v1/jobs/tasks/callback", j.baseURL)
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	resp, err := j.httpClient.Post(url, "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to call back task status: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to call back task status: invalid status code: %d", resp.StatusCode)
	}
	var respBody types.CommonResp
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &respBody, nil
}
