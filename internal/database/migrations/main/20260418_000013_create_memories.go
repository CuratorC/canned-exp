package migrations

import "github.com/CuratorC/gocanned/migration"

// CreateMemories 创建 Agent 长期记忆表
type CreateMemories struct{}

func (m *CreateMemories) Up() string {
	return `
CREATE TABLE IF NOT EXISTS memories (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    agent_id    INTEGER NOT NULL,
    path        TEXT    NOT NULL,
    title       TEXT    NOT NULL DEFAULT '',
    content     TEXT    NOT NULL,
    memory_date TEXT    NOT NULL DEFAULT '',
    created_at  TEXT    NOT NULL,
    updated_at  TEXT    NOT NULL,
    deleted_at  TEXT    DEFAULT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_memories_agent_path ON memories(agent_id, path) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_memories_agent_date ON memories(agent_id, memory_date) WHERE deleted_at IS NULL;
`
}

func (m *CreateMemories) Down() string {
	return `
DROP TABLE IF EXISTS memories;
`
}

func init() {
	migration.Register("main", &CreateMemories{})
}
