package renderer

import (
	"fmt"
	"sort"
	"strings"
)

// RenderedFile holds one rendered config file's content
type RenderedFile struct {
	ConfigPath string
	Content    string
}

// FrameworkRenderer renders personality data into framework-specific config files
type FrameworkRenderer interface {
	Framework() string
	Render(agentName string, personalities map[string]string) []RenderedFile
}

// Registry holds all registered renderers, keyed by framework name
type Registry struct {
	renderers map[string]FrameworkRenderer
}

// NewRegistry creates a Registry with the default renderers
func NewRegistry() *Registry {
	r := &Registry{renderers: make(map[string]FrameworkRenderer)}
	r.Register(&OpenClawRenderer{})
	r.Register(&ClaudeCodeRenderer{})
	return r
}

// Register adds a renderer to the registry
func (r *Registry) Register(renderer FrameworkRenderer) {
	r.renderers[renderer.Framework()] = renderer
}

// Get returns the renderer for the given framework, or nil
func (r *Registry) Get(framework string) FrameworkRenderer {
	return r.renderers[framework]
}

// Render renders personality data for the given framework.
// Falls back to a generic renderer for unknown frameworks.
func (r *Registry) Render(framework, agentName string, personalities map[string]string) []RenderedFile {
	if renderer := r.Get(framework); renderer != nil {
		return renderer.Render(agentName, personalities)
	}
	return fallbackRender(framework, agentName, personalities)
}

func fallbackRender(framework, agentName string, personalities map[string]string) []RenderedFile {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# %s Configuration for %s\n\n", framework, agentName))

	keys := sortedKeys(personalities)
	for _, key := range keys {
		b.WriteString(fmt.Sprintf("## %s\n\n%s\n\n", key, personalities[key]))
	}

	return []RenderedFile{
		{ConfigPath: "CONFIG.md", Content: b.String()},
	}
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
