package model

import (
	"strings"
	"testing"
)

func TestPersonality_Validation(t *testing.T) {
	v := GetValidator()

	t.Run("合法输入通过校验", func(t *testing.T) {
		p := Personality{
			AgentID: 1,
			KeyID:   1,
			Value:   "你好",
			Type:    "string",
		}
		if err := v.Struct(p); err != nil {
			t.Fatalf("expected validation to pass, got error: %v", err)
		}
	})

	t.Run("空 agent_id 被拒绝", func(t *testing.T) {
		p := Personality{
			AgentID: 0,
			KeyID:   1,
			Value:   "有值",
		}
		if err := v.Struct(p); err == nil {
			t.Fatal("expected error for zero agent_id, got nil")
		}
	})

	t.Run("空 key_id 被拒绝", func(t *testing.T) {
		p := Personality{
			AgentID: 1,
			KeyID:   0,
			Value:   "有值",
		}
		if err := v.Struct(p); err == nil {
			t.Fatal("expected error for zero key_id, got nil")
		}
	})

	t.Run("空 value 被拒绝", func(t *testing.T) {
		p := Personality{
			AgentID: 1,
			KeyID:   1,
			Value:   "",
		}
		if err := v.Struct(p); err == nil {
			t.Fatal("expected error for empty value, got nil")
		}
	})

	t.Run("纯空格 value 被拒绝", func(t *testing.T) {
		p := Personality{
			AgentID: 1,
			KeyID:   1,
			Value:   "   \t\n  ",
		}
		if err := v.Struct(p); err == nil {
			t.Fatal("expected error for whitespace-only value, got nil")
		}
	})

	t.Run("无效 type 被拒绝", func(t *testing.T) {
		p := Personality{
			AgentID: 1,
			KeyID:   1,
			Value:   "有值",
			Type:    "invalid",
		}
		if err := v.Struct(p); err == nil {
			t.Fatal("expected error for invalid type, got nil")
		}
	})

	t.Run("有效 type 值通过", func(t *testing.T) {
		for _, typ := range []string{"string", "text", "number", "boolean"} {
			p := Personality{
				AgentID: 1,
				KeyID:   1,
				Value:   "有值",
				Type:    typ,
			}
			if err := v.Struct(p); err != nil {
				t.Fatalf("expected type %q to pass, got error: %v", typ, err)
			}
		}
	})

	t.Run("type 为空时通过校验", func(t *testing.T) {
		p := Personality{
			AgentID: 1,
			KeyID:   1,
			Value:   "有值",
			Type:    "",
		}
		if err := v.Struct(p); err != nil {
			t.Fatalf("expected empty type to pass, got error: %v", err)
		}
	})
}

func TestPersonality_BeforeCreate_DefaultType(t *testing.T) {
	t.Run("空 type 默认设为 string", func(t *testing.T) {
		p := &Personality{
			AgentID: 1,
			KeyID:   1,
			Value:   "测试",
			Type:    "",
		}
		_ = p.BeforeCreate(nil)
		if p.Type != "string" {
			t.Errorf("expected Type to default to 'string', got %q", p.Type)
		}
	})

	t.Run("已有 type 不被覆盖", func(t *testing.T) {
		p := &Personality{
			AgentID: 1,
			KeyID:   1,
			Value:   "测试",
			Type:    "number",
		}
		_ = p.BeforeCreate(nil)
		if p.Type != "number" {
			t.Errorf("expected Type to remain 'number', got %q", p.Type)
		}
	})
}

// 超长 value 拒绝（如果未来加了 trimmedMax 限制）
func TestPersonality_ValueLength(t *testing.T) {
	v := GetValidator()

	t.Run("超长 value 当前不拒绝（无长度限制）", func(t *testing.T) {
		p := Personality{
			AgentID: 1,
			KeyID:   1,
			Value:   strings.Repeat("很长的值", 1000),
		}
		if err := v.Struct(p); err != nil {
			t.Fatalf("expected long value to pass (no length limit yet), got error: %v", err)
		}
	})
}
