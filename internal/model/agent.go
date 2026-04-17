package model

import (
	"github.com/dromara/carbon/v2"
	"gorm.io/gorm"
)

// Agent 代表一个 AI Agent，拥有独立的经验空间
type Agent struct {
	ID          uint          `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string        `json:"name" binding:"trimmedRequired,trimmedMax=100" validate:"trimmedRequired,trimmedMax=100" gorm:"not null"`
	Description string        `json:"description" gorm:"not null;default:''"`
	CreatedAt   carbon.Carbon `json:"created_at"`
	UpdatedAt   carbon.Carbon `json:"updated_at"`
}

// BeforeCreate GORM hook — 创建前自动设置时间戳
func (a *Agent) BeforeCreate(tx *gorm.DB) error {
	now := *carbon.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	return nil
}

// BeforeUpdate GORM hook — 更新前自动刷新时间戳
func (a *Agent) BeforeUpdate(tx *gorm.DB) error {
	a.UpdatedAt = *carbon.Now()
	return nil
}
