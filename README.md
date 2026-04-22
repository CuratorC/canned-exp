# canned-exp

通用 AI Agent 经验库与人格管理 MCP Server，提供经验存储、语义检索、人格配置、记忆管理等能力，支持多 Agent、多平台。

## 特性

- **MCP 协议**：支持 SSE + Streamable HTTP 双传输，兼容 Claude Code、Hermes 等主流 Agent
- **经验库（Experience）**：向量语义检索的知识片段，去重检测
- **人格管理（Personality）**：EAV 模式存储，按框架渲染为配置文件（OpenClaw、Claude Code 等）
- **记忆系统（Memory）**：按路径/日期组织的长期内容，支持路径前缀过滤和日期范围查询
- **REST 端点**：`/api/search` 供 Hook 脚本、自动化工具直接调用
- **TOTP 认证**：分层权限控制，本地回环免认证
- **轻量存储**：SQLite 单文件，零外部依赖

## 快速开始

### 前置条件

- Go 1.26+
- 一个兼容 OpenAI Embedding API 的服务（如智谱、DeepSeek）

### 构建

```bash
go build -o bin/canned-exp ./main.go
```

### 配置

在项目根目录创建 `.env` 文件：

```env
# 应用
APP_ENV=local
APP_PORT=3100

# 数据库（默认 SQLite）
DB_NAME=storage/experience.db

# 向量嵌入（必填）
EMBEDDING_PROVIDER=zhipu
EMBEDDING_API_KEY=<your-api-key>
EMBEDDING_BASE_URL=https://open.bigmodel.cn/api/paas/v4/embeddings
EMBEDDING_MODEL=embedding-3
EMBEDDING_DIMENSIONS=2048

# 认证（可选）
EXPERIENCE_TOTP_SECRET=          # TOTP 共享密钥（base32）
EXPERIENCE_SESSION_TTL=24h       # Session 有效期
```

### 运行

```bash
# 初始化数据库
./bin/canned-exp migrate run

# 启动服务（HTTP + MCP SSE + MCP Streamable HTTP）
./bin/canned-exp serve
```

## 接入 Agent

### Claude Code（SSE 传输）

在 `~/.claude.json` 的 `mcpServers` 中添加：

```json
{
  "canned-exp": {
    "type": "sse",
    "url": "http://localhost:3100/sse"
  }
}
```

### Hermes / 其他 Agent（Streamable HTTP 传输）

配置 MCP 连接地址为：`http://localhost:3100/mcp`

> 注意：如果系统设置了代理，需配置 `NO_PROXY=localhost,127.0.0.1` 排除本地地址。

## API 端点

| 端点 | 方法 | 认证 | 描述 |
|------|------|------|------|
| `/health` | GET | 免认证 | 健康检查 |
| `/health-check` | GET | 免认证 | 健康检查（兼容旧端点） |
| `/sse` | GET | Token / 本地 | MCP SSE 连接 |
| `/message` | POST | Token / 本地 | MCP SSE 消息 |
| `/mcp` | POST | Token / 本地 | MCP Streamable HTTP |
| `/api/search` | POST | Token / 本地 | REST 语义搜索 |
| `/api/auth/login` | POST | 免认证 | TOTP 登录 |
| `/api/auth/revoke` | POST | Token | 吊销 session |

## MCP Tools（24 个）

### Experience（经验库）

| 工具 | 类型 | 描述 |
|------|------|------|
| `save_experience` | 写入 | 保存经验，自动生成向量，支持去重检测 |
| `search_experiences` | 只读 | 语义检索经验 |
| `get_experience` | 只读 | 通过 ID 获取经验 |
| `update_experience` | 写入 | 更新经验，自动重新嵌入 |
| `delete_experience` | 破坏性 | 删除经验 |
| `list_experiences` | 只读 | 分页列出经验 |

### Agent

| 工具 | 类型 | 描述 |
|------|------|------|
| `register_agent` | 写入 | 注册新 Agent |
| `list_agents` | 只读 | 列出所有 Agent |
| `get_agent` | 只读 | 获取 Agent 信息 |
| `update_agent` | 写入 | 更新 Agent |
| `delete_agent` | 破坏性 | 删除 Agent |

### Personality（人格管理）

| 工具 | 类型 | 描述 |
|------|------|------|
| `set_personality` | 写入 | 设置人格属性（Upsert 语义） |
| `get_personality` | 只读 | 获取人格属性 |
| `update_personality` | 写入 | 更新人格属性 |
| `delete_personality` | 破坏性 | 删除人格属性 |
| `list_personalities` | 只读 | 列出 Agent 的所有人格属性 |
| `list_personality_keys` | 只读 | 列出所有可用属性 Key |
| `register_personality_key` | 写入 | 注册新属性 Key |
| `render_personality` | 只读 | 渲染为框架配置文件（OpenClaw / Claude Code） |

### Memory（记忆系统）

| 工具 | 类型 | 描述 |
|------|------|------|
| `save_memory` | 写入 | 保存记忆（Upsert：同 agent_id+path 更新） |
| `get_memory` | 只读 | 按路径获取记忆 |
| `list_memories` | 只读 | 列出记忆（路径前缀 / 日期范围过滤） |
| `search_memories` | 只读 | 文本搜索记忆（标题+内容） |
| `delete_memory` | 破坏性 | 删除记忆 |

## 认证

| 层级 | 认证方式 | 权限范围 |
|------|---------|---------|
| 本地访问 | 127.0.0.1 自动放行 | 全部功能 |
| Session Token | TOTP 登录获取 | 全部功能 |

```bash
# 登录
curl -X POST http://localhost:3100/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"totp_code":"123456","service":"myhost:MyAgent"}'
```

## 配置项

| 环境变量 | 默认值 | 描述 |
|----------|--------|------|
| `APP_ENV` | `prod` | 环境：local / dev / prod |
| `APP_PORT` | `3000` | 服务端口 |
| `DB_DRIVER` | `sqlite` | 数据库驱动 |
| `DB_NAME` | `storage/experience.db` | SQLite 文件路径 |
| `EMBEDDING_PROVIDER` | `zhipu` | 嵌入服务提供商 |
| `EMBEDDING_API_KEY` | （空） | 嵌入 API 密钥 |
| `EMBEDDING_MODEL` | `embedding-3` | 嵌入模型 |
| `EMBEDDING_DIMENSIONS` | `2048` | 向量维度 |
| `EXPERIENCE_TOTP_SECRET` | （空） | TOTP 密钥 |
| `EXPERIENCE_SESSION_TTL` | `24h` | Session 有效期 |

## 开发

```bash
go test ./...                          # 运行测试
go build -o bin/canned-exp ./main.go   # 构建
./bin/canned-exp serve                 # 启动服务
./bin/canned-exp migrate status        # 查看迁移状态
```

## License

MIT
