package migrations

import "github.com/CuratorC/gocanned/migration"

type CreateVectors struct{}

func (m *CreateVectors) Up() string {
	return `
CREATE TABLE IF NOT EXISTS vectors (
    id TEXT PRIMARY KEY,
    vector BLOB NOT NULL
);
`
}

func (m *CreateVectors) Down() string {
	return `DROP TABLE IF EXISTS vectors;`
}

func init() {
	migration.Register("main", &CreateVectors{})
}
