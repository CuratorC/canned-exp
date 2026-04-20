package renderer

import (
	"strings"
	"testing"
)

func TestClaudeCode_FullPersonality(t *testing.T) {
	r := &ClaudeCodeRenderer{}
	p := map[string]string{
		"system_prompt":    "你是一个助手",
		"name":             "测试助手",
		"creature":         "AI",
		"vibe":             "友好",
		"emoji":            "✨",
		"avatar":           "https://example.com/avatar.png",
		"appearance":       "蓝色光芒",
		"language_style":   "简洁",
		"user_naming":      "用户",
		"startup_flow":     "读取配置",
		"daily_routines":   "定时任务",
		"behavioral_rules": "安全规则",
		"relationship_context": "助手关系",
		"tool_preferences": "偏好工具A",
		"environment_config": "生产环境",
		"user_profile":     "技术用户",
	}

	files := r.Render("TestAgent", p)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].ConfigPath != "CLAUDE.md" {
		t.Errorf("expected CLAUDE.md, got %s", files[0].ConfigPath)
	}

	content := files[0].Content
	for _, key := range claudeCodeOrderedKeys {
		if !strings.Contains(content, "## "+key) {
			t.Errorf("expected section for key %q", key)
		}
	}
}

func TestClaudeCode_PartialPersonality(t *testing.T) {
	r := &ClaudeCodeRenderer{}
	p := map[string]string{
		"system_prompt": "你是一个助手",
		"name":          "测试",
	}

	files := r.Render("TestAgent", p)
	content := files[0].Content
	if !strings.Contains(content, "## system_prompt") {
		t.Error("expected system_prompt section")
	}
	if !strings.Contains(content, "## name") {
		t.Error("expected name section")
	}
	if strings.Contains(content, "## creature") {
		t.Error("should not contain creature section for empty value")
	}
}

func TestClaudeCode_Ordering(t *testing.T) {
	r := &ClaudeCodeRenderer{}
	p := map[string]string{
		"user_profile":       "last",
		"system_prompt":      "first",
		"tool_preferences":   "mid",
	}

	files := r.Render("", p)
	content := files[0].Content
	idxSP := strings.Index(content, "## system_prompt")
	idxTP := strings.Index(content, "## tool_preferences")
	idxUP := strings.Index(content, "## user_profile")
	if idxSP >= idxTP || idxTP >= idxUP {
		t.Error("expected ordered sections")
	}
}

func TestClaudeCode_UnknownKeys(t *testing.T) {
	r := &ClaudeCodeRenderer{}
	p := map[string]string{
		"system_prompt": "core",
		"custom_field":  "custom value",
	}

	files := r.Render("", p)
	content := files[0].Content
	idxSP := strings.Index(content, "## system_prompt")
	idxCustom := strings.Index(content, "## custom_field")
	if idxCustom <= idxSP {
		t.Error("expected custom keys after known keys")
	}
}

func TestClaudeCode_Empty(t *testing.T) {
	r := &ClaudeCodeRenderer{}
	files := r.Render("EmptyAgent", map[string]string{})
	content := files[0].Content
	if !strings.Contains(content, "# CLAUDE.md") {
		t.Error("expected CLAUDE.md header")
	}
	if !strings.Contains(content, "EmptyAgent") {
		t.Error("expected agent name in output")
	}
}
