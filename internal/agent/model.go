package agent

import (
	"fmt"
	"strings"

	"github.com/dromara/carbon/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Agent 代表一个 AI Agent，拥有独立的经验空间
type Agent struct {
	ID          string        `json:"id" gorm:"primaryKey"`
	Name        string        `json:"name" gorm:"not null"`
	Description string        `json:"description" gorm:"not null;default:''"`
	CreatedAt   carbon.Carbon `json:"created_at"`
	UpdatedAt   carbon.Carbon `json:"updated_at"`
}

// BeforeCreate GORM hook — 创建前自动设置 ID 和时间戳
func (a *Agent) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
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

// Validate 检查 Agent 必填字段是否合法
func (a *Agent) Validate() error {
	name := strings.TrimSpace(a.Name)
	if name == "" {
		return fmt.Errorf("agent name cannot be empty")
	}
	if len(name) > 100 {
		return fmt.Errorf("agent name exceeds maximum length of 100 characters")
	}
	return nil
}
