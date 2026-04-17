package model

import (
	"strings"
	"testing"
)

func TestFrameworkMapping_Validation(t *testing.T) {
	v := GetValidator()

	t.Run("合法输入通过", func(t *testing.T) {
		m := FrameworkMapping{
			KeyID:      1,
			Framework:  "openclaw",
			ConfigPath: "SOUL.md",
		}
		if err := v.Struct(m); err != nil {
			t.Fatalf("expected pass, got: %v", err)
		}
	})

	t.Run("空 key_id 被拒绝", func(t *testing.T) {
		m := FrameworkMapping{
			KeyID:      0,
			Framework:  "openclaw",
			ConfigPath: "SOUL.md",
		}
		if err := v.Struct(m); err == nil {
			t.Fatal("expected error for zero key_id")
		}
	})

	t.Run("空 framework 被拒绝", func(t *testing.T) {
		m := FrameworkMapping{
			KeyID:      1,
			Framework:  "",
			ConfigPath: "SOUL.md",
		}
		if err := v.Struct(m); err == nil {
			t.Fatal("expected error for empty framework")
		}
	})

	t.Run("纯空格 framework 被拒绝", func(t *testing.T) {
		m := FrameworkMapping{
			KeyID:      1,
			Framework:  "   ",
			ConfigPath: "SOUL.md",
		}
		if err := v.Struct(m); err == nil {
			t.Fatal("expected error for whitespace framework")
		}
	})

	t.Run("空 config_path 被拒绝", func(t *testing.T) {
		m := FrameworkMapping{
			KeyID:      1,
			Framework:  "openclaw",
			ConfigPath: "",
		}
		if err := v.Struct(m); err == nil {
			t.Fatal("expected error for empty config_path")
		}
	})

	t.Run("framework 超长被拒绝", func(t *testing.T) {
		m := FrameworkMapping{
			KeyID:      1,
			Framework:  strings.Repeat("a", 51),
			ConfigPath: "file.md",
		}
		if err := v.Struct(m); err == nil {
			t.Fatal("expected error for framework exceeding 50 chars")
		}
	})

	t.Run("framework 恰好 50 字符通过", func(t *testing.T) {
		m := FrameworkMapping{
			KeyID:      1,
			Framework:  strings.Repeat("a", 50),
			ConfigPath: "file.md",
		}
		if err := v.Struct(m); err != nil {
			t.Fatalf("expected pass for 50 chars, got: %v", err)
		}
	})
}
