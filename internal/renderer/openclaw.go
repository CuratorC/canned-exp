package renderer

import (
	"fmt"
	"strings"
)

// OpenClawRenderer renders personality data into OpenClaw config files
type OpenClawRenderer struct{}

func (r *OpenClawRenderer) Framework() string { return "openclaw" }

func (r *OpenClawRenderer) Render(agentName string, personalities map[string]string) []RenderedFile {
	return []RenderedFile{
		r.renderSOUL(agentName, personalities),
		r.renderIDENTITY(personalities),
		r.renderAGENTS(personalities),
		r.renderMEMORY(agentName, personalities),
		r.renderTOOLS(personalities),
		r.renderUSER(personalities),
	}
}

func (r *OpenClawRenderer) renderSOUL(agentName string, p map[string]string) RenderedFile {
	var b strings.Builder
	b.WriteString("# SOUL.md\n\n")

	if sp, ok := p["system_prompt"]; ok {
		if name, hasName := p["name"]; hasName {
			b.WriteString(fmt.Sprintf("## Persona: %s\n\n", name))
		} else if agentName != "" {
			b.WriteString(fmt.Sprintf("## Persona: %s\n\n", agentName))
		}
		b.WriteString(sp)
		b.WriteString("\n\n")
	}

	if ls, ok := p["language_style"]; ok {
		b.WriteString("## 语言风格\n\n")
		b.WriteString(ls)
		b.WriteString("\n\n")
	}

	return RenderedFile{ConfigPath: "SOUL.md", Content: b.String()}
}

func (r *OpenClawRenderer) renderIDENTITY(p map[string]string) RenderedFile {
	var b strings.Builder
	b.WriteString("# IDENTITY.md\n\n")

	if name, ok := p["name"]; ok {
		b.WriteString(fmt.Sprintf("- **Name:** %s\n", name))
	}
	if creature, ok := p["creature"]; ok {
		b.WriteString(fmt.Sprintf("- **Creature:** %s\n", creature))
	}
	if vibe, ok := p["vibe"]; ok {
		b.WriteString(fmt.Sprintf("- **Vibe:** %s\n", vibe))
	}
	if emoji, ok := p["emoji"]; ok {
		b.WriteString(fmt.Sprintf("- **Emoji:** %s\n", emoji))
	}
	if avatar, ok := p["avatar"]; ok {
		b.WriteString(fmt.Sprintf("- **Avatar:** %s\n", avatar))
	}

	if un, ok := p["user_naming"]; ok {
		b.WriteString(fmt.Sprintf("- **称呼:** %s\n", un))
	}

	if appearance, ok := p["appearance"]; ok {
		b.WriteString("\n## 外貌特征\n\n")
		b.WriteString(appearance)
		b.WriteString("\n")
	}

	return RenderedFile{ConfigPath: "IDENTITY.md", Content: b.String()}
}

func (r *OpenClawRenderer) renderAGENTS(p map[string]string) RenderedFile {
	var b strings.Builder
	b.WriteString("# AGENTS.md\n\n")

	if sf, ok := p["startup_flow"]; ok {
		b.WriteString("## Session Startup\n\n")
		b.WriteString(sf)
		b.WriteString("\n\n")
	}

	return RenderedFile{ConfigPath: "AGENTS.md", Content: b.String()}
}

func (r *OpenClawRenderer) renderMEMORY(agentName string, p map[string]string) RenderedFile {
	var b strings.Builder
	if agentName != "" {
		b.WriteString(fmt.Sprintf("# %s的长期记忆\n\n", agentName))
	} else {
		b.WriteString("# 长期记忆\n\n")
	}

	if dr, ok := p["daily_routines"]; ok {
		b.WriteString("## 日常固定流程\n\n")
		b.WriteString(dr)
		b.WriteString("\n\n")
	}

	if br, ok := p["behavioral_rules"]; ok {
		b.WriteString("## 行为规则\n\n")
		b.WriteString(br)
		b.WriteString("\n\n")
	}

	if rc, ok := p["relationship_context"]; ok {
		b.WriteString("## 关系背景\n\n")
		b.WriteString(rc)
		b.WriteString("\n\n")
	}

	return RenderedFile{ConfigPath: "MEMORY.md", Content: b.String()}
}

func (r *OpenClawRenderer) renderTOOLS(p map[string]string) RenderedFile {
	var b strings.Builder
	b.WriteString("# TOOLS.md\n\n")

	if tp, ok := p["tool_preferences"]; ok {
		b.WriteString("## 工具偏好\n\n")
		b.WriteString(tp)
		b.WriteString("\n\n")
	}

	if ec, ok := p["environment_config"]; ok {
		b.WriteString("## 环境配置\n\n")
		b.WriteString(ec)
		b.WriteString("\n\n")
	}

	return RenderedFile{ConfigPath: "TOOLS.md", Content: b.String()}
}

func (r *OpenClawRenderer) renderUSER(p map[string]string) RenderedFile {
	var b strings.Builder
	b.WriteString("# USER.md\n\n")

	if up, ok := p["user_profile"]; ok {
		b.WriteString("## 用户档案\n\n")
		b.WriteString(up)
		b.WriteString("\n\n")
	}

	return RenderedFile{ConfigPath: "USER.md", Content: b.String()}
}
