package renderer

import (
	"strings"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r.Get("openclaw") == nil {
		t.Error("expected openclaw renderer")
	}
	if r.Get("claude-code") == nil {
		t.Error("expected claude-code renderer")
	}
	if r.Get("unknown") != nil {
		t.Error("expected nil for unknown framework")
	}
}

func TestRegistry_Fallback(t *testing.T) {
	r := NewRegistry()
	files := r.Render("unknown-fw", "TestAgent", map[string]string{
		"custom_key": "custom value",
	})
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].ConfigPath != "CONFIG.md" {
		t.Errorf("expected CONFIG.md, got %s", files[0].ConfigPath)
	}
	if !strings.Contains(files[0].Content, "custom_key") {
		t.Error("expected custom_key in output")
	}
	if !strings.Contains(files[0].Content, "custom value") {
		t.Error("expected custom value in output")
	}
}

func TestRegistry_EmptyPersonality(t *testing.T) {
	r := NewRegistry()
	files := r.Render("claude-code", "TestAgent", map[string]string{})
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
}
