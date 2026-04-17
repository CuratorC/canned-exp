package model

import (
	"strings"
	"testing"
)

func TestPersonalityKey_Validation(t *testing.T) {
	v := GetValidator()

	t.Run("合法输入通过验证", func(t *testing.T) {
		pk := PersonalityKey{
			KeyName:     "system_prompt",
			Description: "核心人格描述",
		}
		if err := v.Struct(pk); err != nil {
			t.Fatalf("expected validation to pass, got error: %v", err)
		}
	})

	t.Run("空 key_name 被拒绝", func(t *testing.T) {
		pk := PersonalityKey{KeyName: "", Description: "has description"}
		err := v.Struct(pk)
		if err == nil {
			t.Fatal("expected error for empty key_name, got nil")
		}
	})

	t.Run("纯空格 key_name 被拒绝", func(t *testing.T) {
		pk := PersonalityKey{KeyName: "   \t\n", Description: "has description"}
		err := v.Struct(pk)
		if err == nil {
			t.Fatal("expected error for whitespace-only key_name, got nil")
		}
	})

	t.Run("key_name 超长被拒绝", func(t *testing.T) {
		pk := PersonalityKey{KeyName: strings.Repeat("a", 101), Description: ""}
		err := v.Struct(pk)
		if err == nil {
			t.Fatal("expected error for key_name exceeding 100 chars, got nil")
		}
	})

	t.Run("key_name 恰好 100 字符通过", func(t *testing.T) {
		pk := PersonalityKey{KeyName: strings.Repeat("a", 100), Description: ""}
		if err := v.Struct(pk); err != nil {
			t.Fatalf("expected pass for exactly 100 chars, got error: %v", err)
		}
	})

	t.Run("空 description 允许", func(t *testing.T) {
		pk := PersonalityKey{KeyName: "test_key", Description: ""}
		if err := v.Struct(pk); err != nil {
			t.Fatalf("expected pass with empty description, got error: %v", err)
		}
	})

	t.Run("含特殊字符的 key_name 通过", func(t *testing.T) {
		pk := PersonalityKey{KeyName: "my-key.v2", Description: ""}
		if err := v.Struct(pk); err != nil {
			t.Fatalf("expected pass for key_name with special chars, got error: %v", err)
		}
	})
}
