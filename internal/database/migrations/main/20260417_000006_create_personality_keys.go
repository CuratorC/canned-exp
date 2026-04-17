package migrations

import "github.com/CuratorC/gocanned/migration"

// CreatePersonalityKeys 创建人格属性 key 枚举表 + 预置记录
type CreatePersonalityKeys struct{}

func (m *CreatePersonalityKeys) Up() string {
	return `
CREATE TABLE IF NOT EXISTS personality_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key_name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

INSERT INTO personality_keys (key_name, description, created_at, updated_at) VALUES
    ('system_prompt', '核心人格描述', datetime('now'), datetime('now')),
    ('name', '人格名称', datetime('now'), datetime('now')),
    ('creature', '本质/物种描述', datetime('now'), datetime('now')),
    ('vibe', '风格关键词', datetime('now'), datetime('now')),
    ('emoji', '标志性 emoji', datetime('now'), datetime('now')),
    ('avatar', '头像 URL', datetime('now'), datetime('now'));
`
}

func (m *CreatePersonalityKeys) Down() string {
	return `DROP TABLE IF EXISTS personality_keys;`
}

func init() {
	migration.Register("main", &CreatePersonalityKeys{})
}
