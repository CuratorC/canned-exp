package model

import (
	"strings"
	"testing"
)

func TestAgent_Validation(t *testing.T) {
	v := GetValidator()

	t.Run("正常Agent通过验证", func(t *testing.T) {
		a := Agent{
			Name:        "Claude",
			Description: "An AI coding assistant",
		}
		if err := v.Struct(a); err != nil {
			t.Fatalf("expected validation to pass, got error: %v", err)
		}
	})

	t.Run("空名称被拒绝", func(t *testing.T) {
		a := Agent{Name: "", Description: "has description"}
		err := v.Struct(a)
		if err == nil {
			t.Fatal("expected error for empty name, got nil")
		}
	})

	t.Run("纯空格名称被拒绝", func(t *testing.T) {
		a := Agent{Name: "   \t\n", Description: "has description"}
		err := v.Struct(a)
		if err == nil {
			t.Fatal("expected error for whitespace-only name, got nil")
		}
	})

	t.Run("超长名称被拒绝", func(t *testing.T) {
		a := Agent{Name: strings.Repeat("a", 101), Description: "has description"}
		err := v.Struct(a)
		if err == nil {
			t.Fatal("expected error for name exceeding 100 chars, got nil")
		}
	})

	t.Run("空描述通过", func(t *testing.T) {
		a := Agent{Name: "TestAgent", Description: ""}
		if err := v.Struct(a); err != nil {
			t.Fatalf("expected pass with empty description, got error: %v", err)
		}
	})

	t.Run("名称恰好100字符通过", func(t *testing.T) {
		a := Agent{Name: strings.Repeat("a", 100), Description: ""}
		if err := v.Struct(a); err != nil {
			t.Fatalf("expected pass for exactly 100 chars, got error: %v", err)
		}
	})
}
