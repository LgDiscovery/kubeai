package model

import (
	"database/sql/driver"
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
	StatusSubmitted TaskStatus = "submitted"
	StatusQueued    TaskStatus = "queued"
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

func (r ResourceRequest) Value() (driver.Value, error) {
	return json.Marshal(r)
}

func (r *ResourceRequest) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b := value.([]byte)
	return json.Unmarshal(b, r)
}

type JSONMap map[string]interface{}

func (m JSONMap) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b := value.([]byte)
	return json.Unmarshal(b, m)
}

// InferenceTask 推理任务数据库模型
type InferenceTask struct {
	ID            int64           `gorm:"primaryKey;autoIncrement" json:"-"`
	TaskID        string          `gorm:"column:task_id;type:varchar(64);uniqueIndex;not null" json:"task_id"`
	Name          string          `gorm:"column:name;type:varchar(255);uniqueIndex;not null" json:"name"`
	ModelName     string          `gorm:"column:model_name;type:varchar(255);not null" json:"model_name"`
	ModelVersion  string          `gorm:"column:model_version;type:varchar(50);not null" json:"model_version"`
	ModelPath     string          `gorm:"column:model_path;type:text" json:"model_path"`
	Framework     string          `gorm:"column:framework;type:varchar(50)" json:"framework"`
	Resources     ResourceRequest `gorm:"column:resources;type:jsonb;not null" json:"resources"`
	InputData     JSONMap         `gorm:"column:input_data;type:jsonb" json:"input_data"`
	OutputTopic   string          `gorm:"column:output_topic;type:varchar(255)" json:"output_topic"`
	Status        TaskStatus      `gorm:"column:status;type:varchar(20);default:pending;index" json:"status"`
	Priority      int             `gorm:"column:priority;default:5" json:"priority"`
	RetryCount    int             `gorm:"column:retry_count;default:0" json:"retry_count"`
	MaxRetries    int             `gorm:"column:max_retries;default:3" json:"max_retries"`
	ScheduledNode string          `gorm:"column:scheduled_node;type:varchar(255)" json:"scheduled_node"`
	PodName       string          `gorm:"column:pod_name;type:varchar(255)" json:"pod_name"`
	ErrorMessage  string          `gorm:"column:error_message;type:text" json:"error_message"`
	CreatedAt     time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (t *InferenceTask) Marshal() ([]byte, error) {
	return json.Marshal(t)
}

func UnmarshalInferenceTask(data []byte) (*InferenceTask, error) {
	var t InferenceTask
	err := json.Unmarshal(data, &t)
	return &t, err
}

type StringArray []string

func (a StringArray) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b := value.([]byte)
	return json.Unmarshal(b, a)
}

// TrainingTask 训练任务
type TrainingTask struct {
	ID            int64           `gorm:"primaryKey;autoIncrement" json:"-"`
	RequestID     string          `gorm:"column:request_id;type:varchar(64);uniqueIndex;not null" json:"request_id"`
	TaskID        string          `gorm:"column:task_id;type:varchar(64);uniqueIndex;not null" json:"task_id"`
	Name          string          `gorm:"column:name;type:varchar(255);uniqueIndex;not null" json:"name"`
	ModelName     string          `gorm:"column:model_name;type:varchar(255)" json:"model_name"`
	Framework     string          `gorm:"column:framework;type:varchar(50);not null" json:"framework"`
	Image         string          `gorm:"column:image;type:varchar(255);not null" json:"image"`
	Command       StringArray     `gorm:"column:command;type:jsonb;not null" json:"command"`
	Args          StringArray     `gorm:"column:args;type:jsonb" json:"args"`
	Distributed   bool            `gorm:"column:distributed;default:false" json:"distributed"` // 新增
	WorkerNum     int32           `gorm:"column:worker_num;default:1" json:"worker_num"`       // 新增
	MasterNum     int32           `gorm:"column:master_num;default:1" json:"master_num"`       // 新增
	Env           []EnvVar        `gorm:"column:env;type:jsonb" json:"env"`
	Resources     ResourceRequest `gorm:"column:resources;type:jsonb;not null" json:"resources"`
	DatasetPath   string          `gorm:"column:dataset_path;type:text" json:"dataset_path"`
	OutputPath    string          `gorm:"column:output_path;type:text" json:"output_path"`
	Status        TaskStatus      `gorm:"column:status;type:varchar(20);default:pending;index" json:"status"`
	Priority      int             `gorm:"column:priority;default:5" json:"priority"`
	RetryCount    int             `gorm:"column:retry_count;default:0" json:"retry_count"`
	MaxRetries    int             `gorm:"column:max_retries;default:3" json:"max_retries"`
	ScheduledNode string          `gorm:"column:scheduled_node;type:varchar(255)" json:"scheduled_node"`
	Volumes       []Volume        `gorm:"column:volumes;type:jsonb;not null" json:"volume"`
	VolumeMounts  []VolumeMount   `gorm:"column:mount_volumes;type:jsonb;not null" json:"mount_volume"`
	PodName       string          `gorm:"column:pod_name;type:varchar(255)" json:"pod_name"`
	ErrorMessage  string          `gorm:"column:error_message;type:text" json:"error_message"`
	CreatedAt     time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	EnableMonitor bool            `gorm:"column:enable_monitor;default:false" json:"enable_monitor"`
	EnableLogs    bool            `gorm:"column:enable_logs;default:false" json:"enable_logs"`
}

type Volume struct {
	Name      string `json:"name"`
	ClaimName string `json:"claim_name"`
}

type VolumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mount_path"`
}

func (t *TrainingTask) Marshal() ([]byte, error) {
	return json.Marshal(t)
}

func UnmarshalTrainingTask(data []byte) (*TrainingTask, error) {
	var t TrainingTask
	err := json.Unmarshal(data, &t)
	return &t, err
}

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
