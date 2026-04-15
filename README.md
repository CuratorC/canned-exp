# canned-exp

通用个人经验库 MCP Server，为 AI Agent 提供经验的长期存储、语义检索与优化能力，支持跨平台、跨应用的经验共享。

## 特性

- **MCP 协议**：通过 SSE 传输，Claude Code、Cursor 等主流 Agent 原生支持
- **REST 端点**：提供 `/api/search` 供 Hook 脚本、自动化工具直接调用
- **TOTP 认证**：支持 Google Authenticator 等标准验证器，分层权限控制
- **语义检索**：基于向量嵌入的语义搜索，而非简单关键词匹配
- **OpenAI 兼容**：支持任何兼容 OpenAI Embedding API 的平台（OpenAI、DeepSeek、本地 Ollama 等）
- **轻量存储**：SQLite 单文件存储，无需额外数据库服务
- **6 个核心工具**：save / search / get / update / delete / list
- **工具注解**：MCP 协议标准注解，区分只读/写入/破坏性操作

## 架构

```
main.go
  └─> canned-exp mcp serve           — MCP Server（SSE + REST + Auth）
       └─> bootstrap.SetupExperience()
            ├─> SQLite 数据库（storage/experiences.db）
            ├─> 自动运行迁移
            ├─> Embedding HTTP 客户端
            └─> Auth 认证管理器
                 ├─> TOTP 验证
                 ├─> Session 管理（内存，按服务隔离）
                 └─> API Key 校验
```

```
canned-exp/
├── cmd/
│   ├── serve.go                   # HTTP Server（原有）
│   └── mcp.go                     # MCP Server 子命令（SSE + REST + Auth）
├── config/
│   ├── embedding.go               # Embedding API 配置
│   └── experience.go              # 经验库配置（数据库、MCP 端口、认证）
├── bootstrap/
│   ├── bootstrap.go               # HTTP 应用引导
│   └── experience.go              # 经验库应用引导
├── internal/
│   ├── experience/
│   │   ├── model.go               # 数据模型
│   │   ├── repository.go          # SQLite CRUD + 向量检索
│   │   └── service.go             # 业务逻辑（去重、搜索、更新向量）
│   └── experience/mcp/
│       ├── server.go              # 工具定义 + 调度（6 个工具 + 注解）
│       ├── transport.go           # SSE + REST 组合服务器（CombinedServer）
│       ├── auth.go                # TOTP 认证、Session 管理、中间件
│       └── rest.go                # REST 端点（/api/search）
└── .env                           # 环境变量配置
```

## 快速开始

### 前置条件

- Go 1.23+
- 一个兼容 OpenAI Embedding API 的服务

### 构建

```bash
go build -o bin/canned-exp ./main.go
```

### 配置

在项目根目录创建 `.env` 文件：

```env
# Embedding 服务配置
EMBEDDING_BASE_URL=https://api.openai.com/v1
EMBEDDING_API_KEY=sk-xxx
EMBEDDING_MODEL=text-embedding-3-small
EMBEDDING_DIMENSIONS=1536

# 数据库
MCP_DB_NAME=storage/experiences.db

# MCP Server
EXPERIENCE_MCP_PORT=3100

# 认证（可选，不配置则无认证保护）
EXPERIENCE_TOTP_SECRET=          # TOTP 共享密钥（base32）
EXPERIENCE_API_KEY=              # 静态 API Key（用于 Hook 脚本只读访问）
EXPERIENCE_SESSION_TTL=24h       # Session 有效期
```

### 运行

```bash
./bin/canned-exp mcp serve
```

### 接入 Agent

在 Claude Code 的 MCP 配置（`~/.claude.json`）中添加：

```json
{
  "mcpServers": {
    "canned-exp": {
      "type": "sse",
      "url": "http://localhost:3100/sse",
      "headers": {
        "Authorization": "Bearer <session-token>"
      }
    }
  }
}
```

## 认证系统

### 三层权限

| 层级 | 认证方式 | 权限范围 |
|------|---------|---------|
| 本地访问 | 127.0.0.1 自动放行 | 全部功能 |
| API Key | `Authorization: Bearer <api_key>` | 仅 `/api/search`（只读） |
| Session Token | TOTP 登录获取 | 全部功能 |

### API 端点

| 端点 | 方法 | 认证 | 描述 |
|------|------|------|------|
| `/sse` | GET | Token / 本地 | MCP SSE 连接 |
| `/message` | POST | Token / 本地 | MCP 消息 |
| `/api/search` | POST | Token / API Key / 本地 | REST 搜索（供 Hook 使用） |
| `/api/auth/login` | POST | 免认证 | TOTP 登录获取 session |
| `/api/auth/revoke` | POST | 由 handler 校验 Token | 吊销指定服务的 session |

### 登录与续期

```bash
# 登录（需要 TOTP 验证码）
curl -X POST http://localhost:3100/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"totp_code":"123456","service":"myhost:MyAgent"}'

# 吊销服务 session
curl -X POST http://localhost:3100/api/auth/revoke \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"service":"myhost:OtherAgent"}'
```

- `service` 参数格式：`主机名:工具名`，确保同一工具始终使用相同 service
- 每个 service 仅保留最新 session，新登录自动吊销旧 session
- Session 存储在内存中，服务重启后全部失效

## MCP Tools

| 工具 | 类型 | 描述 | 参数 |
|------|------|------|------|
| `save_experience` | 写入 | 保存一条经验，自动生成向量嵌入 | `content`（必填）, `tags`（必填）, `title`, `source`, `force` |
| `search_experiences` | 只读 | 语义检索经验 | `query`（必填）, `top_k`（默认 5） |
| `get_experience` | 只读 | 通过 ID 获取经验详情 | `id`（必填） |
| `update_experience` | 写入 | 更新经验，内容变更时自动重新嵌入 | `id`（必填）, `content`, `title`, `tags` |
| `delete_experience` | 破坏性 | 删除经验 | `id`（必填） |
| `list_experiences` | 只读 | 分页列出所有经验 | `page`（默认 1）, `pageSize`（默认 20） |

### 使用示例

Agent 保存一条经验：

```
save_experience(
  content = "在 Go 中使用 context.WithTimeout 时，应该始终在 defer 中调用 cancel()，即使 timeout 已经过期。",
  tags = ["go", "context", "best-practice"]
)
```

Agent 语义检索经验：

```
search_experiences(
  query = "Go context 使用注意事项",
  top_k = 3
)
```

## 配置项

| 环境变量 | 默认值 | 描述 |
|----------|--------|------|
| `EMBEDDING_BASE_URL` | `https://api.openai.com/v1` | Embedding API 地址 |
| `EMBEDDING_API_KEY` | （空） | Embedding API 密钥 |
| `EMBEDDING_MODEL` | `text-embedding-3-small` | Embedding 模型名称 |
| `EMBEDDING_DIMENSIONS` | `1536` | 向量维度 |
| `MCP_DB_NAME` | `storage/experiences.db` | SQLite 数据库文件路径 |
| `EXPERIENCE_MCP_PORT` | `3100` | MCP Server 监听端口 |
| `EXPERIENCE_TOTP_SECRET` | （空） | TOTP 共享密钥 |
| `EXPERIENCE_API_KEY` | （空） | 静态 API Key |
| `EXPERIENCE_SESSION_TTL` | `24h` | Session 有效期 |

## 技术细节

### 向量存储

- 向量以 `[]float32` 序列化为 BLOB 存储在 SQLite 中（每维度 4 字节，1536 维 ≈ 6KB/条）
- 检索时加载全部向量，在内存中计算余弦相似度
- MVP 阶段适合 < 10K 条经验，毫秒级响应

### 数据库

使用独立的 SQLite 文件，不依赖 MySQL 或 Redis，MCP Server 可独立部署。

经验表结构：

```sql
CREATE TABLE experiences (
    id         TEXT PRIMARY KEY,
    title      TEXT NOT NULL DEFAULT '',
    content    TEXT NOT NULL,
    tags       TEXT NOT NULL DEFAULT '',
    source     TEXT NOT NULL DEFAULT '',
    embedding  BLOB,
    created_at DATETIME,
    updated_at DATETIME
);
```

### 认证实现

- **TOTP**：使用 `github.com/pquerna/otp/totp`，兼容 Google Authenticator
- **Session**：内存 `sync.Map` 存储，服务重启即失效，无外部依赖
- **API Key**：静态比对，适用于 Hook 脚本等自动化场景

## 开发

```bash
# 运行测试
go test ./...

# 构建
go build -o bin/canned-exp ./main.go

# 运行 MCP Server
./bin/canned-exp mcp serve

# 运行 HTTP Server（原有功能）
./bin/canned-exp serve
```

## License

MIT
