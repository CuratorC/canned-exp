package experience

import (
	"fmt"
	"strings"
)

// MaxContentLength 经验内容的最大字符数限制（约 8K tokens）
const MaxContentLength = 8000

// Experience 代表一条经验记录
type Experience struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Tags      []string `json:"tags"`
	Source    string   `json:"source"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

// Validate 校验经验内容是否合法
func (e *Experience) Validate() error {
	content := strings.TrimSpace(e.Content)
	if content == "" {
		return fmt.Errorf("content cannot be empty")
	}
	if len(content) > MaxContentLength {
		return fmt.Errorf("content exceeds maximum length of %d characters", MaxContentLength)
	}
	return nil
}

// TextToEmbed 返回用于向量化的文本（title + content 拼接）
// 前提：调用方已通过 Validate 校验
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
