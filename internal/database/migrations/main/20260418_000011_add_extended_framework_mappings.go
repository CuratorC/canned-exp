package migrations

import "github.com/CuratorC/gocanned/migration"

// AddExtendedFrameworkMappings 追加扩展 key 的 OpenClaw 和 Claude Code 映射
// key_id 7-16 对应 000010 迁移新增的 personality_keys
type AddExtendedFrameworkMappings struct{}

func (m *AddExtendedFrameworkMappings) Up() string {
	return `
-- OpenClaw 扩展映射（key_id 7-16）
INSERT INTO framework_mappings (key_id, framework, config_path, created_at, updated_at) VALUES
    (7,  'openclaw', 'IDENTITY.md', datetime('now'), datetime('now')),
    (8,  'openclaw', 'SOUL.md', datetime('now'), datetime('now')),
    (9,  'openclaw', 'IDENTITY.md', datetime('now'), datetime('now')),
    (10, 'openclaw', 'AGENTS.md', datetime('now'), datetime('now')),
    (11, 'openclaw', 'MEMORY.md', datetime('now'), datetime('now')),
    (12, 'openclaw', 'MEMORY.md', datetime('now'), datetime('now')),
    (13, 'openclaw', 'MEMORY.md', datetime('now'), datetime('now')),
    (14, 'openclaw', 'TOOLS.md', datetime('now'), datetime('now')),
    (15, 'openclaw', 'TOOLS.md', datetime('now'), datetime('now')),
    (16, 'openclaw', 'USER.md', datetime('now'), datetime('now'));

-- Claude Code 扩展映射
INSERT INTO framework_mappings (key_id, framework, config_path, created_at, updated_at) VALUES
    (7,  'claude-code', 'CLAUDE.md', datetime('now'), datetime('now')),
    (8,  'claude-code', 'CLAUDE.md', datetime('now'), datetime('now')),
    (9,  'claude-code', 'CLAUDE.md', datetime('now'), datetime('now')),
    (10, 'claude-code', 'CLAUDE.md', datetime('now'), datetime('now')),
    (11, 'claude-code', 'CLAUDE.md', datetime('now'), datetime('now')),
    (12, 'claude-code', 'CLAUDE.md', datetime('now'), datetime('now')),
    (13, 'claude-code', 'CLAUDE.md', datetime('now'), datetime('now')),
    (14, 'claude-code', 'CLAUDE.md', datetime('now'), datetime('now')),
    (15, 'claude-code', 'CLAUDE.md', datetime('now'), datetime('now')),
    (16, 'claude-code', 'CLAUDE.md', datetime('now'), datetime('now'));
`
}

func (m *AddExtendedFrameworkMappings) Down() string {
	return `
DELETE FROM framework_mappings WHERE key_id BETWEEN 7 AND 16;
`
}

func init() {
	migration.Register("main", &AddExtendedFrameworkMappings{})
}
