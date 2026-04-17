package model

import (
	"github.com/dromara/carbon/v2"
	"gorm.io/gorm"
)

// Personality Agent 的人格属性（EAV 模式）
// 每个 Agent 可以有多条 Personality，每条对应一个 PersonalityKey
type Personality struct {
	ID        uint          `json:"id" gorm:"primaryKey;autoIncrement"`
	AgentID   uint          `json:"agent_id" validate:"required" gorm:"index;not null"`
	KeyID     uint          `json:"key_id" validate:"required" gorm:"not null"`
	Value     string        `json:"value" validate:"trimmedRequired"`
	Type      string        `json:"type" validate:"personalityType" gorm:"not null;default:'string'"`
	CreatedAt carbon.Carbon `json:"created_at"`
	UpdatedAt carbon.Carbon `json:"updated_at"`
}

func (p *Personality) BeforeCreate(tx *gorm.DB) error {
	if p.Type == "" {
		p.Type = "string"
	}
	now := *carbon.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	return nil
}

func (p *Personality) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = *carbon.Now()
	return nil
}
