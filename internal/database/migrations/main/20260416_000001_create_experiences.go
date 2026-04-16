package migrations

import "github.com/CuratorC/gocanned/migration"

type CreateExperiences struct{}

func (m *CreateExperiences) Up() string {
	return `
CREATE TABLE IF NOT EXISTS experiences (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    tags TEXT NOT NULL DEFAULT '[]',
    source TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
`
}

func (m *CreateExperiences) Down() string {
	return `DROP TABLE IF EXISTS experiences;`
}

func init() {
	migration.Register("main", &CreateExperiences{})
}
