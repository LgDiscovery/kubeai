package model

import (
	"time"

	"gorm.io/gorm"
)

// User 用户
type User struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	Username  string         `gorm:"size:50;not null;uniqueIndex" json:"username"`
	Password  string         `gorm:"size:50;not null" json:"password"`
	Role      string         `gorm:"size:50;default:'user'" json:"role"`
	Status    string         `gorm:"size:20;default:'active'" json:"status"` // active/staged/archived
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
