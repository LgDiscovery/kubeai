package model

import (
	"time"

	"gorm.io/gorm"
)

// Model 模型元数据表
type Model struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Name        string         `gorm:"uniqueIndex;size:255;not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Framework   string         `gorm:"size:50" json:"framework"` // pytorch/tensorflow/onnx
	TaskType    string         `gorm:"size:50" json:"task_type"` // classification/regression/llm
	Owner       string         `gorm:"size:100" json:"owner"`
	Labels      string         `gorm:"type:jsonb" json:"labels"` // JSON 格式标签
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Versions    []ModelVersion `gorm:"foreignKey:ModelID" json:"versions,omitempty"`
}

// ModelVersion 模型版本表
type ModelVersion struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	ModelID      uint           `gorm:"not null;index" json:"model_id"`
	Version      string         `gorm:"size:50;not null;uniqueIndex:idx_model_version" json:"version"`
	Description  string         `gorm:"type:text" json:"description"`
	StoragePath  string         `gorm:"size:500;not null" json:"storage_path"` // MinIO 对象路径
	Framework    string         `gorm:"size:50" json:"framework"`
	FrameworkVer string         `gorm:"size:20" json:"framework_version"`
	Metrics      string         `gorm:"type:jsonb" json:"metrics"` // 评估指标 JSON
	Parameters   string         `gorm:"type:jsonb" json:"parameters"`
	Size         int64          `json:"size"`                                   // 文件大小(字节)
	Checksum     string         `gorm:"size:128" json:"checksum"`               // SHA256
	Status       string         `gorm:"size:20;default:'active'" json:"status"` // active/staged/archived
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Model        Model          `gorm:"foreignKey:ModelID" json:"model,omitempty"`
}
