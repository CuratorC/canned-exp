package migrations

import "github.com/CuratorC/gocanned/migration"

// AddSoftDelete 为 agents 和 personalities 添加软删除支持
// personalities 的唯一索引改为 partial index（仅对未删除记录生效）
type AddSoftDelete struct{}

func (m *AddSoftDelete) Up() string {
	return `
-- 1. agents: 添加 deleted_at 列
ALTER TABLE agents ADD COLUMN deleted_at TEXT;

-- 2. personalities: 添加 deleted_at 列
ALTER TABLE personalities ADD COLUMN deleted_at TEXT;

-- 3. personalities: 唯一索引改为 partial index
--    先删除旧索引，再创建新的 partial index
DROP INDEX IF EXISTS idx_personalities_agent_key;
CREATE UNIQUE INDEX idx_personalities_agent_key ON personalities(agent_id, key_id) WHERE deleted_at IS NULL;
`
}

func (m *AddSoftDelete) Down() string {
	return `
-- Down 不提供，partial index 转换不可逆
`
}

func init() {
	migration.Register("main", &AddSoftDelete{})
}
