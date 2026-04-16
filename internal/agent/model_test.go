package agent

import (
	"strings"
	"testing"
)

func TestAgent_Validate(t *testing.T) {
	t.Run("正常Agent通过验证", func(t *testing.T) {
		a := &Agent{
			Name:        "Claude",
			Description: "An AI coding assistant",
		}
		if err := a.Validate(); err != nil {
			t.Fatalf("expected validation to pass, got error: %v", err)
		}
	})

	t.Run("空名称被拒绝", func(t *testing.T) {
		a := &Agent{
			Name:        "",
			Description: "has description",
		}
		err := a.Validate()
		if err == nil {
			t.Fatal("expected error for empty name, got nil")
		}
		if !strings.Contains(err.Error(), "name") {
			t.Errorf("error should mention 'name', got: %v", err)
		}
	})

	t.Run("纯空格名称被拒绝", func(t *testing.T) {
		a := &Agent{
			Name:        "   \t\n",
			Description: "has description",
		}
		err := a.Validate()
		if err == nil {
			t.Fatal("expected error for whitespace-only name, got nil")
		}
	})

	t.Run("超长名称被拒绝", func(t *testing.T) {
		a := &Agent{
			Name:        strings.Repeat("a", 101),
			Description: "has description",
		}
		err := a.Validate()
		if err == nil {
			t.Fatal("expected error for name exceeding 100 chars, got nil")
		}
		if !strings.Contains(err.Error(), "100") {
			t.Errorf("error should mention length limit '100', got: %v", err)
		}
	})

	t.Run("空描述是允许的", func(t *testing.T) {
		a := &Agent{
			Name:        "TestAgent",
			Description: "",
		}
		if err := a.Validate(); err != nil {
			t.Fatalf("expected validation to pass with empty description, got error: %v", err)
		}
	})

	t.Run("名称恰好100字符通过", func(t *testing.T) {
		a := &Agent{
			Name:        strings.Repeat("a", 100),
			Description: "",
		}
		if err := a.Validate(); err != nil {
			t.Fatalf("expected name of exactly 100 chars to pass, got error: %v", err)
		}
	})
}
