package model

import (
	"github.com/dromara/carbon/v2"
	"gorm.io/gorm"
)

// Memory Agent 的长期记忆，按路径和日期组织
type Memory struct {
	ID         uint          `json:"id" gorm:"primaryKey;autoIncrement"`
	AgentID    uint          `json:"agent_id" validate:"required" gorm:"index;not null"`
	Path       string        `json:"path" validate:"trimmedRequired,trimmedMax=200" gorm:"not null"`
	Title      string        `json:"title" gorm:"not null;default:''"`
	Content    string        `json:"content" validate:"trimmedRequired" gorm:"not null"`
	MemoryDate string        `json:"memory_date" gorm:"not null;default:''"`
	CreatedAt  carbon.Carbon `json:"created_at"`
	UpdatedAt  carbon.Carbon `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (m *Memory) BeforeCreate(tx *gorm.DB) error {
	now := *carbon.Now()
	m.CreatedAt = now
	m.UpdatedAt = now
	return nil
}

func (m *Memory) BeforeUpdate(tx *gorm.DB) error {
	m.UpdatedAt = *carbon.Now()
	return nil
}
