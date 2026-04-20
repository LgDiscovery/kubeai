package client

import (
	"fmt"
	"net/http"
	"time"
)

type InferenceGatewayClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewInferenceGatewayClient(baseURL string, timeout time.Duration) *InferenceGatewayClient {
	return &InferenceGatewayClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *InferenceGatewayClient) PauseTrainingTask(taskID string) error {
	url := fmt.Sprintf("%s/api/v1/trainingjob/%s/pause", c.baseURL, taskID)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pause task failed, status code: %d", resp.StatusCode)
	}
	return nil
}

func (c *InferenceGatewayClient) ResumeTrainingTask(taskID string) error {
	url := fmt.Sprintf("%s/api/v1/trainingjob/%s/resume", c.baseURL, taskID)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("resume task failed, status code: %d", resp.StatusCode)
	}
	return nil
}

func (c *InferenceGatewayClient) CancelTrainingTask(taskID string) error {
	url := fmt.Sprintf("%s/api/v1/trainingjob/%s/cancel", c.baseURL, taskID)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cancel task failed, status code: %d", resp.StatusCode)
	}
	return nil
}
