package migrations

import "github.com/CuratorC/gocanned/migration"

// AddExtendedPersonalityKeys 追加扩展人格属性 key
type AddExtendedPersonalityKeys struct{}

func (m *AddExtendedPersonalityKeys) Up() string {
	return `
INSERT INTO personality_keys (key_name, description, created_at, updated_at) VALUES
    ('appearance', '外貌特征描述', datetime('now'), datetime('now')),
    ('language_style', '语言风格与口癖', datetime('now'), datetime('now')),
    ('user_naming', '对用户的称呼方式', datetime('now'), datetime('now')),
    ('startup_flow', '启动/初始化流程', datetime('now'), datetime('now')),
    ('daily_routines', '日常固定流程', datetime('now'), datetime('now')),
    ('behavioral_rules', '行为规则与边界', datetime('now'), datetime('now')),
    ('relationship_context', '与用户的关系背景', datetime('now'), datetime('now')),
    ('tool_preferences', '工具偏好与配置', datetime('now'), datetime('now')),
    ('environment_config', '环境配置（代理、API等）', datetime('now'), datetime('now')),
    ('user_profile', '用户档案信息', datetime('now'), datetime('now'));
`
}

func (m *AddExtendedPersonalityKeys) Down() string {
	return `
DELETE FROM personality_keys WHERE key_name IN (
    'appearance', 'language_style', 'user_naming', 'startup_flow',
    'daily_routines', 'behavioral_rules', 'relationship_context',
    'tool_preferences', 'environment_config', 'user_profile'
);
`
}

func init() {
	migration.Register("main", &AddExtendedPersonalityKeys{})
}
