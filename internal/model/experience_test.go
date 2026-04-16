package model

import (
	"encoding/json"
	"strings"
	"testing"
)

// --- Validate 测试 ---

func TestExperience_Validate(t *testing.T) {
	t.Run("正常经验通过校验", func(t *testing.T) {
		exp := Experience{
			Title:   "Go TDD 测试文件组织",
			Content: "测试文件要和源文件同目录，包名用 xxx_test 可以避免循环导入",
		}
		if err := exp.Validate(); err != nil {
			t.Errorf("期望通过校验，但得到: %v", err)
		}
	})

	t.Run("无标题时通过校验", func(t *testing.T) {
		exp := Experience{
			Content: "经验内容是核心，标题可选",
		}
		if err := exp.Validate(); err != nil {
			t.Errorf("期望通过校验，但得到: %v", err)
		}
	})

	t.Run("content 为空时拒绝", func(t *testing.T) {
		exp := Experience{Title: "有标题无内容"}
		if err := exp.Validate(); err == nil {
			t.Error("期望返回错误，但得到了 nil")
		}
	})

	t.Run("content 仅空白字符时拒绝", func(t *testing.T) {
		exp := Experience{
			Title:   "有标题",
			Content: "   \t\n  ",
		}
		if err := exp.Validate(); err == nil {
			t.Error("期望返回错误，但得到了 nil")
		}
	})

	t.Run("content 超长时拒绝", func(t *testing.T) {
		exp := Experience{
			Title:   "超长经验",
			Content: strings.Repeat("这是一段很长的经验内容。", 2000),
		}
		if err := exp.Validate(); err == nil {
			t.Error("期望返回超长错误，但得到了 nil")
		}
	})
}

// --- TextToEmbed 测试 ---
// 前提：调用方已通过 Validate 校验，此处只测拼接格式

func TestExperience_TextToEmbed(t *testing.T) {
	t.Run("有 title 和 content 时用双换行拼接", func(t *testing.T) {
		exp := Experience{
			Title:   "Go TDD 测试文件组织",
			Content: "测试文件要和源文件同目录",
		}
		got := exp.TextToEmbed()
		want := "Go TDD 测试文件组织\n\n测试文件要和源文件同目录"
		if got != want {
			t.Errorf("TextToEmbed() = %q, want %q", got, want)
		}
	})

	t.Run("无 title 时直接返回 content", func(t *testing.T) {
		exp := Experience{Content: "没有标题的经验"}
		got := exp.TextToEmbed()
		if got != "没有标题的经验" {
			t.Errorf("TextToEmbed() = %q, want %q", got, "没有标题的经验")
		}
	})
}

// --- AgentID 序列化测试 ---

func TestExperience_AgentID_Serialization(t *testing.T) {
	t.Run("AgentID非空时序列化包含该字段", func(t *testing.T) {
		exp := Experience{
			AgentID: "agent-123",
			Content: "test content",
		}
		data, err := json.Marshal(exp)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
		got := string(data)
		if !strings.Contains(got, `"agent_id":"agent-123"`) {
			t.Errorf("expected JSON to contain agent_id, got: %s", got)
		}
	})

	t.Run("AgentID为空时序列化省略该字段", func(t *testing.T) {
		exp := Experience{
			AgentID: "",
			Content: "test content",
		}
		data, err := json.Marshal(exp)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
		got := string(data)
		if strings.Contains(got, "agent_id") {
			t.Errorf("expected agent_id to be omitted when empty, got: %s", got)
		}
	})

	t.Run("AgentID正确反序列化", func(t *testing.T) {
		jsonStr := `{"id":"exp-1","agent_id":"agent-456","content":"hello"}`
		var exp Experience
		if err := json.Unmarshal([]byte(jsonStr), &exp); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if exp.AgentID != "agent-456" {
			t.Errorf("AgentID mismatch: got %q, want %q", exp.AgentID, "agent-456")
		}
	})

	t.Run("缺少agent_id时反序列化为空字符串", func(t *testing.T) {
		jsonStr := `{"id":"exp-2","content":"no agent"}`
		var exp Experience
		if err := json.Unmarshal([]byte(jsonStr), &exp); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if exp.AgentID != "" {
			t.Errorf("AgentID should be empty, got %q", exp.AgentID)
		}
	})
}
