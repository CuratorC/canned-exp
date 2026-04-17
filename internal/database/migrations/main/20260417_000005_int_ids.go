package migrations

import "github.com/CuratorC/gocanned/migration"

// ConvertAgentToIntID 将 agents.id 从 TEXT UUID 转为 INTEGER 自增
// 同时将 experiences.agent_id 从 TEXT 转为 INTEGER
// SQLite 不支持 ALTER COLUMN，agents 需要重建表，experiences.agent_id 也需要重建表
type ConvertAgentToIntID struct{}

func (m *ConvertAgentToIntID) Up() string {
	return `
-- 1. agents: TEXT id → INTEGER id
CREATE TABLE agents_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
INSERT INTO agents_new (name, description, created_at, updated_at) SELECT name, description, created_at, updated_at FROM agents;
DROP TABLE agents;
ALTER TABLE agents_new RENAME TO agents;

-- 2. experiences: agent_id TEXT → INTEGER
-- 注意：旧 agent_id 是 UUID string，无法映射到新的 int id，重置为 0（全局）
CREATE TABLE experiences_new (
    id TEXT PRIMARY KEY,
    agent_id INTEGER NOT NULL DEFAULT 0,
    title TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    tags TEXT NOT NULL DEFAULT '[]',
    source TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
INSERT INTO experiences_new (id, agent_id, title, content, tags, source, created_at, updated_at)
    SELECT id, 0, title, content, tags, source, created_at, updated_at FROM experiences;
DROP TABLE experiences;
ALTER TABLE experiences_new RENAME TO experiences;
CREATE INDEX IF NOT EXISTS idx_experiences_agent_id ON experiences(agent_id);
`
}

func (m *ConvertAgentToIntID) Down() string {
	return `
-- Down 不提供，int ID 转换不可逆
`
}

func init() {
	migration.Register("main", &ConvertAgentToIntID{})
}
