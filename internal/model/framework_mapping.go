package model

import (
	"github.com/dromara/carbon/v2"
	"gorm.io/gorm"
)

// FrameworkMapping 记录 PersonalityKey 在不同框架中的配置位置映射
type FrameworkMapping struct {
	ID         uint          `json:"id" gorm:"primaryKey;autoIncrement"`
	KeyID      uint          `json:"key_id" validate:"required" gorm:"not null"`
	Framework  string        `json:"framework" validate:"trimmedRequired,trimmedMax=50" gorm:"not null"`
	ConfigPath string        `json:"config_path" validate:"trimmedRequired,trimmedMax=200" gorm:"not null"`
	CreatedAt  carbon.Carbon `json:"created_at"`
	UpdatedAt  carbon.Carbon `json:"updated_at"`
}

func (m *FrameworkMapping) BeforeCreate(tx *gorm.DB) error {
	now := *carbon.Now()
	m.CreatedAt = now
	m.UpdatedAt = now
	return nil
}

func (m *FrameworkMapping) BeforeUpdate(tx *gorm.DB) error {
	m.UpdatedAt = *carbon.Now()
	return nil
}
