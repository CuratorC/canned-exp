package migrations

import "github.com/CuratorC/gocanned/migration"

// CreateFrameworkMappings 创建框架映射表 + 预置 OpenClaw 和 Claude Code 映射
type CreateFrameworkMappings struct{}

func (m *CreateFrameworkMappings) Up() string {
	return `
CREATE TABLE IF NOT EXISTS framework_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key_id INTEGER NOT NULL,
    framework TEXT NOT NULL,
    config_path TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_framework_mappings_key_framework ON framework_mappings(key_id, framework);
CREATE INDEX IF NOT EXISTS idx_framework_mappings_framework ON framework_mappings(framework);

-- OpenClaw 映射（key_id 1-6 对应预置的 personality_keys）
INSERT INTO framework_mappings (key_id, framework, config_path, created_at, updated_at) VALUES
    (1, 'openclaw', 'SOUL.md', datetime('now'), datetime('now')),
    (2, 'openclaw', 'IDENTITY.md', datetime('now'), datetime('now')),
    (3, 'openclaw', 'IDENTITY.md', datetime('now'), datetime('now')),
    (4, 'openclaw', 'IDENTITY.md', datetime('now'), datetime('now')),
    (5, 'openclaw', 'IDENTITY.md', datetime('now'), datetime('now')),
    (6, 'openclaw', 'IDENTITY.md', datetime('now'), datetime('now'));

-- Claude Code 映射
INSERT INTO framework_mappings (key_id, framework, config_path, created_at, updated_at) VALUES
    (1, 'claude-code', 'CLAUDE.md', datetime('now'), datetime('now')),
    (2, 'claude-code', 'CLAUDE.md', datetime('now'), datetime('now'));
`
}

func (m *CreateFrameworkMappings) Down() string {
	return `DROP TABLE IF EXISTS framework_mappings;`
}

func init() {
	migration.Register("main", &CreateFrameworkMappings{})
}
