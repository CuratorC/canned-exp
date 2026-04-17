package model

import (
	"strings"

	"github.com/dromara/carbon/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Experience 代表一条经验记录
type Experience struct {
	ID        string         `json:"id" gorm:"primaryKey"`
	AgentID   uint           `json:"agent_id,omitempty" gorm:"index;not null;default:0"`
	Title     string         `json:"title" gorm:"not null;default:''"`
	Content   string         `json:"content" binding:"trimmedRequired,trimmedMax=8000" validate:"trimmedRequired,trimmedMax=8000" gorm:"not null;default:''"`
	Tags      []string       `json:"tags" gorm:"serializer:sonic;not null;default:'[]'"`
	Source    string         `json:"source" gorm:"not null;default:''"`
	CreatedAt carbon.Carbon  `json:"created_at"`
	UpdatedAt carbon.Carbon  `json:"updated_at"`
}

// BeforeCreate GORM hook — 创建前自动设置 ID 和时间戳
func (e *Experience) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	now := *carbon.Now()
	e.CreatedAt = now
	e.UpdatedAt = now
	return nil
}

// BeforeUpdate GORM hook — 更新前自动刷新时间戳
func (e *Experience) BeforeUpdate(tx *gorm.DB) error {
	e.UpdatedAt = *carbon.Now()
	return nil
}

// TextToEmbed 返回用于向量化的文本（title + content 拼接）
func (e *Experience) TextToEmbed() string {
	title := strings.TrimSpace(e.Title)
	content := strings.TrimSpace(e.Content)
	if title == "" {
		return content
	}
	if content == "" {
		return title
	}
	return title + "\n\n" + content
}
