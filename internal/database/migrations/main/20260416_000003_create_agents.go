package migrations

import "github.com/CuratorC/gocanned/migration"

type CreateAgents struct{}

func (m *CreateAgents) Up() string {
	return `
CREATE TABLE IF NOT EXISTS agents (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
`
}

func (m *CreateAgents) Down() string {
	return `DROP TABLE IF EXISTS agents;`
}

func init() {
	migration.Register("main", &CreateAgents{})
}
