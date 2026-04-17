package migrations

import "github.com/CuratorC/gocanned/migration"

// CreatePersonalities 创建人格属性 EAV 表
// 每个 agent 的每个 key 只能有一条记录（agent_id + key_id 联合唯一）
type CreatePersonalities struct{}

func (m *CreatePersonalities) Up() string {
	return `
CREATE TABLE IF NOT EXISTS personalities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    agent_id INTEGER NOT NULL,
    key_id INTEGER NOT NULL,
    value TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'string',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_personalities_agent_key ON personalities(agent_id, key_id);
CREATE INDEX IF NOT EXISTS idx_personalities_agent_id ON personalities(agent_id);
`
}

func (m *CreatePersonalities) Down() string {
	return `DROP TABLE IF EXISTS personalities;`
}

func init() {
	migration.Register("main", &CreatePersonalities{})
}
