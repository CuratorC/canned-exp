package model

import (
	"github.com/dromara/carbon/v2"
	"gorm.io/gorm"
)

// PersonalityKey 定义人格属性的 key 枚举
// 每个 key 代表一种人格维度（如 name、system_prompt、vibe 等）
type PersonalityKey struct {
	ID          uint          `json:"id" gorm:"primaryKey;autoIncrement"`
	KeyName     string        `json:"key_name" binding:"trimmedRequired,trimmedMax=100" validate:"trimmedRequired,trimmedMax=100" gorm:"uniqueIndex;not null"`
	Description string        `json:"description" gorm:"not null;default:''"`
	CreatedAt   carbon.Carbon `json:"created_at"`
	UpdatedAt   carbon.Carbon `json:"updated_at"`
}

func (pk *PersonalityKey) BeforeCreate(tx *gorm.DB) error {
	now := *carbon.Now()
	pk.CreatedAt = now
	pk.UpdatedAt = now
	return nil
}

func (pk *PersonalityKey) BeforeUpdate(tx *gorm.DB) error {
	pk.UpdatedAt = *carbon.Now()
	return nil
}
