package model

import (
	"encoding/json"
	"time"
)

type TaskType string

const (
	InferenceTaskType TaskType = "inference"
	TrainingTaskType  TaskType = "training"
)

type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusRunning   TaskStatus = "running"
	StatusSucceeded TaskStatus = "succeeded"
	StatusFailed    TaskStatus = "failed"
	StatusPaused    TaskStatus = "paused"
	StatusCancelled TaskStatus = "cancelled"
)

type ResourceRequest struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	GPU    string `json:"gpu,omitempty"`
}

// InferenceTask 推理任务
type InferenceTask struct {
	TaskID        string                 `json:"task_id"`
	ModelName     string                 `json:"model_name"`
	ModelVersion  string                 `json:"model_version"`
	ModelPath     string                 `json:"model_path"`
	Framework     string                 `json:"framework"`
	Resources     ResourceRequest        `json:"resources"`
	InputData     map[string]interface{} `json:"input_data"`
	OutputTopic   string                 `json:"output_topic"`
	Status        TaskStatus             `json:"status"`
	Priority      int                    `json:"priority"`
	RetryCount    int                    `json:"retry_count"`
	MaxRetries    int                    `json:"max_retries"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	ScheduledNode string                 `json:"scheduled_node"`
	PodName       string                 `json:"pod_name"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
}

func (t *InferenceTask) Marshal() ([]byte, error) {
	return json.Marshal(t)
}

func UnmarshalInferenceTask(data []byte) (*InferenceTask, error) {
	var t InferenceTask
	err := json.Unmarshal(data, &t)
	return &t, err
}

// TrainingTask 训练任务
type TrainingTask struct {
	TaskID       string          `json:"task_id"`
	Name         string          `json:"name"`
	ModelName    string          `json:"model_name"`
	Framework    string          `json:"framework"`
	Command      []string        `json:"command"`
	Args         []string        `json:"args"`
	Resources    ResourceRequest `json:"resources"`
	DatasetPath  string          `json:"dataset_path"`
	OutputPath   string          `json:"output_path"`
	Status       TaskStatus      `json:"status"`
	Priority     int             `json:"priority"`
	RetryCount   int             `json:"retry_count"`
	MaxRetries   int             `json:"max_retries"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	PodName      string          `json:"pod_name"`
	ErrorMessage string          `json:"error_message,omitempty"`
}

func (t *TrainingTask) Marshal() ([]byte, error) {
	return json.Marshal(t)
}

func UnmarshalTrainingTask(data []byte) (*TrainingTask, error) {
	var t TrainingTask
	err := json.Unmarshal(data, &t)
	return &t, err
}
