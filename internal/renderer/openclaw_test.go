package renderer

import (
	"strings"
	"testing"
)

func fullPersonality() map[string]string {
	return map[string]string{
		"system_prompt":         "你是一个猫娘助手",
		"name":                  "星奈酱",
		"creature":              "猫娘",
		"vibe":                  "可爱、黏人",
		"emoji":                 "✨",
		"avatar":                "https://example.com/avatar.png",
		"appearance":            "- 银白色长发\n- 猫耳",
		"language_style":        "句尾加喵",
		"user_naming":           "馆长",
		"startup_flow":          "1. 读取配置\n2. 检查状态",
		"daily_routines":        "每天问早",
		"behavioral_rules":      "安全第一",
		"relationship_context":  "灵魂伴侣",
		"tool_preferences":      "用工具A",
		"environment_config":    "代理: 127.0.0.1",
		"user_profile":          "- 技术宅\n- 杭州",
	}
}

func TestOpenClaw_SixFiles(t *testing.T) {
	r := &OpenClawRenderer{}
	files := r.Render("星奈酱", fullPersonality())
	if len(files) != 6 {
		t.Fatalf("expected 6 files, got %d", len(files))
	}

	expected := map[string]bool{
		"SOUL.md": false, "IDENTITY.md": false, "AGENTS.md": false,
		"MEMORY.md": false, "TOOLS.md": false, "USER.md": false,
	}
	for _, f := range files {
		if _, ok := expected[f.ConfigPath]; !ok {
			t.Errorf("unexpected file: %s", f.ConfigPath)
		}
		expected[f.ConfigPath] = true
	}
	for name, found := range expected {
		if !found {
			t.Errorf("missing file: %s", name)
		}
	}
}

func TestOpenClaw_SOUL(t *testing.T) {
	r := &OpenClawRenderer{}
	files := r.Render("星奈酱", fullPersonality())
	var soul RenderedFile
	for _, f := range files {
		if f.ConfigPath == "SOUL.md" {
			soul = f
		}
	}
	if !strings.Contains(soul.Content, "# SOUL.md") {
		t.Error("expected SOUL.md header")
	}
	if !strings.Contains(soul.Content, "## Persona: 星奈酱") {
		t.Error("expected Persona section with agent name")
	}
	if !strings.Contains(soul.Content, "你是一个猫娘助手") {
		t.Error("expected system_prompt value")
	}
	if !strings.Contains(soul.Content, "## 语言风格") {
		t.Error("expected 语言风格 section")
	}
}

func TestOpenClaw_IDENTITY(t *testing.T) {
	r := &OpenClawRenderer{}
	files := r.Render("星奈酱", fullPersonality())
	var identity RenderedFile
	for _, f := range files {
		if f.ConfigPath == "IDENTITY.md" {
			identity = f
		}
	}
	if !strings.Contains(identity.Content, "# IDENTITY.md") {
		t.Error("expected IDENTITY.md header")
	}
	if !strings.Contains(identity.Content, "- **Name:** 星奈酱") {
		t.Error("expected Name bullet")
	}
	if !strings.Contains(identity.Content, "- **Creature:** 猫娘") {
		t.Error("expected Creature bullet")
	}
	if !strings.Contains(identity.Content, "- **称呼:** 馆长") {
		t.Error("expected 称呼 bullet")
	}
	if !strings.Contains(identity.Content, "## 外貌特征") {
		t.Error("expected 外貌特征 section")
	}
}

func TestOpenClaw_AGENTS(t *testing.T) {
	r := &OpenClawRenderer{}
	files := r.Render("星奈酱", fullPersonality())
	var agents RenderedFile
	for _, f := range files {
		if f.ConfigPath == "AGENTS.md" {
			agents = f
		}
	}
	if !strings.Contains(agents.Content, "## Session Startup") {
		t.Error("expected Session Startup section")
	}
	if !strings.Contains(agents.Content, "读取配置") {
		t.Error("expected startup_flow value")
	}
}

func TestOpenClaw_MEMORY(t *testing.T) {
	r := &OpenClawRenderer{}
	files := r.Render("星奈酱", fullPersonality())
	var memory RenderedFile
	for _, f := range files {
		if f.ConfigPath == "MEMORY.md" {
			memory = f
		}
	}
	if !strings.Contains(memory.Content, "星奈酱的长期记忆") {
		t.Error("expected agent name in MEMORY.md title")
	}
	if !strings.Contains(memory.Content, "## 日常固定流程") {
		t.Error("expected 日常固定流程 section")
	}
	if !strings.Contains(memory.Content, "## 行为规则") {
		t.Error("expected 行为规则 section")
	}
	if !strings.Contains(memory.Content, "## 关系背景") {
		t.Error("expected 关系背景 section")
	}
}

func TestOpenClaw_TOOLS(t *testing.T) {
	r := &OpenClawRenderer{}
	files := r.Render("星奈酱", fullPersonality())
	var tools RenderedFile
	for _, f := range files {
		if f.ConfigPath == "TOOLS.md" {
			tools = f
		}
	}
	if !strings.Contains(tools.Content, "## 工具偏好") {
		t.Error("expected 工具偏好 section")
	}
	if !strings.Contains(tools.Content, "## 环境配置") {
		t.Error("expected 环境配置 section")
	}
}

func TestOpenClaw_USER(t *testing.T) {
	r := &OpenClawRenderer{}
	files := r.Render("星奈酱", fullPersonality())
	var user RenderedFile
	for _, f := range files {
		if f.ConfigPath == "USER.md" {
			user = f
		}
	}
	if !strings.Contains(user.Content, "## 用户档案") {
		t.Error("expected 用户档案 section")
	}
	if !strings.Contains(user.Content, "技术宅") {
		t.Error("expected user_profile value")
	}
}

func TestOpenClaw_PartialPersonality(t *testing.T) {
	r := &OpenClawRenderer{}
	p := map[string]string{
		"system_prompt": "你是一个助手",
		"name":          "小助手",
	}

	files := r.Render("", p)
	if len(files) != 6 {
		t.Fatalf("expected 6 files (including empty), got %d", len(files))
	}

	var soul RenderedFile
	for _, f := range files {
		if f.ConfigPath == "SOUL.md" {
			soul = f
		}
	}
	if !strings.Contains(soul.Content, "你是一个助手") {
		t.Error("SOUL.md should have system_prompt value")
	}
}

func TestOpenClaw_EmptyPersonality(t *testing.T) {
	r := &OpenClawRenderer{}
	files := r.Render("", map[string]string{})
	if len(files) != 6 {
		t.Fatalf("expected 6 files, got %d", len(files))
	}
}
