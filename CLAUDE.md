# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

Go binary is at `/home/curatorc/sdk/go1.26.1/bin/go`, add to PATH before running commands:

```bash
export PATH="/home/curatorc/sdk/go1.26.1/bin:$PATH"
go build -o bin/canned-exp ./main.go   # 编译
go run ./main.go serve                 # 运行 serve 命令
go run ./main.go mcp serve             # 运行 MCP Server（SSE + REST）
go test ./...                          # 运行测试
```

本项目基于 `github.com/CuratorC/gocanned` 框架（本地 replace），提供 config、database、cache、logger、limiter、env 等基础能力。

## Architecture

依赖通过 `internal/app.App` 结构体显式注入，方向单向流动：

**main.go -> bootstrap -> route -> handler -> service -> repository -> DB**

```
main.go                  — 入口，注册 cobra 命令，调用 bootstrap
cmd/serve.go             — serve 子命令，组装 App、启动 HTTP Server、优雅关停
cmd/mcp.go               — mcp serve 子命令，启动 MCP Server（SSE + REST + Auth）
bootstrap/               — 初始化引导层，负责创建所有基础设施
  bootstrap.go           — SetupCommand（config+logger）、NewApp（返回 *app.App）
  config.go              — 配置初始化
  database.go            — 数据库连接初始化，返回 *database.DB
  logger.go              — 日志初始化
  cache.go               — Redis 初始化（gocanned 单例管理）
  route.go               — 注册全局中间件、调用路由注册、404 处理
  proxy.go               — 代理服务初始化（通过 PROXY_ENABLED 开关）
config/                  — 配置定义，各文件通过 init() + config.Add() 注册到 gocanned
internal/app/app.go      — App 结构体，持有所有应用级依赖（DB 等），由 bootstrap 创建
internal/enum/           — 枚举/常量定义
internal/http/
  middleware/             — Gin 中间件（Logger、Recovery、LimitIP/LimitPerRoute、Auth）
  response/              — 统一 HTTP 响应处理（JSON、Success、Data、Error、Abort* 等）
internal/route/api.go    — API 路由注册，接收 *app.App 用于依赖注入
internal/proxy/          — Anthropic Messages API 代理层
  types_anthropic.go     — Anthropic 请求/响应类型定义
  types_openai.go        — OpenAI 请求/响应类型定义
  convert.go             — 双向格式转换（请求 + 响应 + 错误）
  stream.go              — SSE 流式代理（OpenAI chunks → Anthropic SSE events）
  handler.go             — Gin handler + ProxyService 服务
internal/experience/     — 经验库核心（模型、仓储、服务）
  model.go               — 数据模型
  repository.go          — SQLite CRUD + 向量检索
  service.go             — 业务逻辑（去重、搜索、更新向量）
internal/experience/mcp/ — MCP Server 层
  server.go              — 工具定义 + 调度（6 个工具）
  transport.go           — SSE + REST 组合服务器（CombinedServer）
  auth.go                — TOTP 认证、Session 管理、中间件
  rest.go                — REST 端点（/api/search，供 Hook 使用）
```

## Key Conventions

- 依赖注入：基础设施通过 `bootstrap.NewApp()` 创建，经 `app.App` 传递到路由/处理层，禁止使用全局变量
- `config/` 下每个文件用 `init()` + `config.Add()` 注册配置项，环境变量通过 `config.Env()` 读取，运行时通过 `config.GetString()` 等获取
- gocanned 框架内部以单例管理 Redis（`cache.Instance()`）和配置（Viper），限流中间件直接使用框架单例
- 数据库连接通过 `database.Connect()` 创建，返回 `*database.DB`（嵌入 `*sql.DB`），支持 MySQL 和 SQLite
- API 响应统一使用 `internal/http/response` 包的函数
- 日志统一使用 `logger.Info/Error/Warn/Debug`， panic 恢复使用 `logger.ErrorAndExit`
- 项目使用 tab 缩进

## MCP Server 架构

```
Client (Claude Code / Hook / curl)
  │
  ├─ SSE (/sse, /message)    → MCP 协议
  ├─ REST (/api/search)      → Hook 脚本搜索
  ├─ Auth (/api/auth/login)  → TOTP 登录获取 session
  ├─ Auth (/api/auth/revoke) → 吊销指定服务的 session
  └─ Proxy (/v1/messages)    → Anthropic → OpenAI 协议转换
  │
  └─ Auth Middleware
       ├─ /api/auth/login, /api/auth/revoke → 免认证
       ├─ 127.0.0.1 (loopback)              → 免认证
       └─ Session Token                     → 全权限
```

### API 代理

`PROXY_ENABLED=true` 时，`/v1/messages` 端点将 Anthropic Messages API 请求转换为 OpenAI Chat Completions API 格式转发给后端：

- **流式**：Claude Code 通过 `Accept: text/event-stream` 声明，代理实时将 OpenAI SSE chunks 转为 Anthropic SSE events（message_start → content_block_start → delta → stop → message_delta → message_stop）
- **非流式**：直接转译 JSON 响应
- **模型映射**：`PROXY_MODEL_MAP` 将 Claude 模型名映射为后端模型名
- **认证**：复用现有 Auth 中间件，本地 loopback 免认证

### 代理配置

| 环境变量 | 描述 |
|----------|------|
| `PROXY_ENABLED` | 是否启用（默认 false，false 时不注册路由） |
| `PROXY_BASE_URL` | 后端 base URL（默认 `https://api.openai.com/v1`） |
| `PROXY_API_KEY` | 后端 API Key |
| `PROXY_MODEL_MAP` | 模型映射 JSON，格式 `{"claude-model":"backend-model"}` |

### 认证配置

| 环境变量 | 描述 |
|----------|------|
| `AUTH_TOTP_SECRET` | TOTP 共享密钥（base32，用 Google Authenticator 扫描） |
| `AUTH_SESSION_TTL` | Session 有效期（默认 24h） |
| `EXPERIENCE_MCP_PORT` | MCP Server 端口（默认 3000） |

### Session 管理策略

- 每个 `service`（格式：`主机名:工具名`）仅保留一个活跃 session
- 新登录自动吊销同服务的旧 session
- 可通过 `/api/auth/revoke` 主动吊销任意服务的 session
- Session 存储在内存中，服务重启后全部失效

## Experience Library (Long-term Memory)

本项目本身就是一个经验库。在工作中使用它自带的工具来积累和检索知识。

### Memory Priority

**MUST** 经验库是唯一的权威知识源。**任何讨论、回答、任务执行前，都必须先调用 `search_experiences`**。禁止仅凭文件记忆或 CLAUDE.md 内容直接作答。文件记忆和 CLAUDE.md 仅作为补充线索，不得替代经验库检索。

### When to Search

- **每次回复前**，只要涉及项目知识、过往约定、技术决策，都必须检索
- 开始任何任务之前
- 遇到错误或异常行为时
- 用户提到具体技术/项目名时
- 感觉解决过类似问题（似曾相识）时
- 用户询问"你还记得..."、"我们之前..."类问题时

### When to Save

- 非显而易见的调试方案（根因 + 修复）
- 文档中没有的项目约定
- 已知问题的 workaround
- "顿悟"时刻 —— 未来能节省时间的发现
- 对之前错误认知的纠正

### When to Update

- 已有经验不完整或部分错误
- 发现了更好的方案

### Tag Conventions

- 3-7 个标签
- 领域在前：`go`、`python`、`mcp`、`docker`、`sqlite`
- 问题模式在后：`error-handling`、`concurrency`、`deployment`
- 复合标签：`go-context`、`mcp-sse`、`sqlite-vector`
- 避免过宽泛标签：`bug`、`fix`、`tip`

### Content Style

- 祈使句/教学体：告诉未来的 Agent 应该怎么做
- 包含：问题背景 → 根因 → 方案 → 为什么有效
- 不要："我试了 X 结果失败了"（日记体）
- 要："当 X 发生时，做 Y，因为 Z"（教学体）
