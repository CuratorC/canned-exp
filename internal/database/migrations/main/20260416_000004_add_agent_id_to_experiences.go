package migrations

import "github.com/CuratorC/gocanned/migration"

type AddAgentIDToExperiences struct{}

func (m *AddAgentIDToExperiences) Up() string {
	return `
ALTER TABLE experiences ADD COLUMN agent_id TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_experiences_agent_id ON experiences(agent_id);
`
}

func (m *AddAgentIDToExperiences) Down() string {
	return `
DROP INDEX IF EXISTS idx_experiences_agent_id;
ALTER TABLE experiences DROP COLUMN agent_id;
`
}

func init() {
	migration.Register("main", &AddAgentIDToExperiences{})
}
