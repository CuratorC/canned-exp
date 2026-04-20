package renderer

import (
	"fmt"
	"strings"
)

// ClaudeCodeRenderer renders personality data into a single CLAUDE.md
type ClaudeCodeRenderer struct{}

var claudeCodeOrderedKeys = []string{
	"system_prompt", "name", "creature", "vibe", "emoji", "avatar",
	"appearance", "language_style", "user_naming", "startup_flow",
	"daily_routines", "behavioral_rules", "relationship_context",
	"tool_preferences", "environment_config", "user_profile",
}

func (r *ClaudeCodeRenderer) Framework() string { return "claude-code" }

func (r *ClaudeCodeRenderer) Render(agentName string, personalities map[string]string) []RenderedFile {
	var b strings.Builder
	b.WriteString("# CLAUDE.md\n\n")

	if agentName != "" {
		b.WriteString(fmt.Sprintf("> Agent: %s\n\n", agentName))
	}

	known := make(map[string]bool, len(claudeCodeOrderedKeys))
	for _, key := range claudeCodeOrderedKeys {
		known[key] = true
		if val, ok := personalities[key]; ok {
			b.WriteString(fmt.Sprintf("## %s\n\n%s\n\n", key, val))
		}
	}

	for _, key := range sortedKeys(personalities) {
		if !known[key] {
			b.WriteString(fmt.Sprintf("## %s\n\n%s\n\n", key, personalities[key]))
		}
	}

	return []RenderedFile{
		{ConfigPath: "CLAUDE.md", Content: b.String()},
	}
}
