# Letta（原 MemGPT）深度调研

> 调研日期：2026-04-13
> 官网：https://www.letta.com
> GitHub：https://github.com/letta-ai/letta
> 许可证：Apache License 2.0
> Stars：~22,000
> 定位：有状态的 AI Agent 运行时平台

## 1. 解决什么问题

Letta 解决的核心问题是：**LLM 本身是无状态的，如何让它变成一个能自主管理记忆的有状态 Agent**。

传统 LLM 的根本限制：
- 每次交互都是孤立的，不具备跨对话记忆
- 上下文窗口有限（即使 200k tokens 也会超出）
- 模型无法主动决定"记住什么"和"忘记什么"
- 现有方案要么手动管理上下文，要么丢失超出窗口的信息

Letta 起源于 UC Berkeley 的 MemGPT 论文（ICML 2024），核心理念是将 **LLM 的上下文窗口类比为操作系统的内存管理**（RAM vs 磁盘），构建了一套虚拟无限记忆系统。

## 2. 工作原理

### 2.1 分层记忆架构

```
┌─────────────────────────────────────────┐
│              LLM 上下文窗口              │
│  ┌───────────┐  ┌───────────┐          │
│  │ 核心记忆    │  │ 消息缓冲区  │  系统提示  │
│  │ (Memory    │  │ (Message  │          │
│  │  Blocks)   │  │  Buffer)  │          │
│  └─────┬─────┘  └─────┬─────┘          │
│        │               │                │
├────────┼───────────────┼────────────────┤
│        ▼  (工具调用)     ▼ (自动持久化)    │
│  ┌───────────┐  ┌───────────┐          │
│  │ 归档记忆    │  │ 回忆记忆    │          │
│  │ (Archival  │  │ (Recall    │          │
│  │  Memory)   │  │  Memory)   │          │
│  └───────────┘  └───────────┘          │
│   外部数据库         对话历史存档          │
└─────────────────────────────────────────┘
```

### 2.2 四层记忆详解

| 层级 | 名称 | 类比 | 说明 |
|------|------|------|------|
| 第一层 | 消息缓冲区 (Message Buffer) | CPU 缓存 | 存储最近对话，窗口满时自动压缩摘要 |
| 第二层 | 核心记忆 (Core Memory) | RAM | 常驻上下文窗口的记忆块，Agent 可自主编辑 |
| 第三层 | 回忆记忆 (Recall Memory) | SSD | 完整对话历史存档，可搜索检索 |
| 第四层 | 归档记忆 (Archival Memory) | 磁盘 | 结构化知识库，支持向量搜索和图遍历 |

### 2.3 Agent 循环

Letta 经历了两代架构演进：

**第一代 (MemGPT)：**
- 所有动作（包括发送消息）都通过工具调用实现
- 通过 `request_heartbeat` 参数控制是否继续执行
- 通过 `thinking` 参数注入链式推理
- 缺点：强依赖模型支持可靠的工具调用

**第二代 (Letta V1)：**
- 废弃 heartbeat 和 `send_message` 工具
- 原生支持模型的内置推理能力（OpenAI Responses API、Anthropic 推理）
- 兼容任何 LLM（不强制要求工具调用支持）
- 为 GPT-5、Claude 4.5 Sonnet 等前沿模型优化

### 2.4 核心概念

- **Memory Blocks**：上下文窗口中的离散功能单元，包含 label、value、description、limit
- **Sleep-Time Compute**：Agent 空闲时异步处理和更新记忆
- **Tool Rules**：约束工具调用模式的规则引擎
- **Skills & Subagents**：预置技能和子 Agent 模式

## 3. 记忆模型

### 3.1 核心记忆 (Core Memory)

- 以 Memory Blocks 形式**常驻上下文窗口**
- 默认两个核心块：
  - `human` 块：用户信息、偏好
  - `persona` 块：Agent 人格、行为准则
- **Agent 通过工具调用自主编辑**（也可由开发者或外部程序编辑）
- 可设为只读
- 所有 Block 持久化到数据库，有唯一 `block_id`
- 支持多 Agent 共享同一个 Memory Block

### 3.2 消息缓冲区 (Message Buffer)

- 维护最近对话的连续线程
- 窗口满时采用**递归摘要**：旧消息与新摘要合并，越老影响越小
- 被驱逐的消息自动持久化到回忆记忆

### 3.3 回忆记忆 (Recall Memory)

- 完整的对话历史持久化存储
- 可通过搜索检索回上下文窗口
- Letta 自动处理持久化

### 3.4 归档记忆 (Archival Memory)

- 经过处理和索引的结构化知识
- 存储在外部数据库中（向量数据库、图数据库等）
- 通过专用工具查询和检索
- 支持向量搜索（embed + query）和图遍历

## 4. API 概览

### 4.1 部署方式

| 方式 | 说明 |
|------|------|
| 托管 API | `https://api.letta.com`，按用量付费 |
| 自部署 | Docker，完全掌控数据 |
| Letta Code CLI | `npm install -g @letta-ai/letta-code`，终端中运行 |

### 4.2 SDK

- **Python**：`pip install letta-client`
- **TypeScript/Node.js**：`npm install @letta-ai/letta-client`
- **Rust**：社区维护的 `letta` crate

### 4.3 核心端点

| 资源 | 主要操作 |
|------|---------|
| Agents | CRUD、发送消息、流式响应、异步执行、compaction、调度、导出/导入 |
| Memory Blocks | CRUD、与 Agent 绑定/解绑、多 Agent 共享 |
| Archives | CRUD、Passages 管理（创建、删除、搜索） |
| Tools | CRUD、与 Agent 绑定/解绑、审批管理、手动执行 |
| MCP Servers | 管理 MCP 服务器及其工具 |
| Conversations | 会话管理、分支(fork)、compaction |
| Models | 模型列表、嵌入模型列表 |
| Runs | 执行追踪（Steps、Metrics、Trace、Usage） |

### 4.4 使用示例

```python
from letta_client import Letta

client = Letta(api_key=os.getenv("LETTA_API_KEY"))

agent = client.agents.create(
    model="openai/gpt-5.2",
    memory_blocks=[
        {"label": "human", "value": "用户名: 张三, 喜欢Python"},
        {"label": "persona", "value": "你是一个有记忆的编程助手"}
    ],
    tools=["web_search", "fetch_webpage"]
)

response = client.agents.messages.create(
    agent_id=agent.id,
    input="你还记得我的名字吗？"
)
```

## 5. 多 Agent 协作

Letta 支持多种多 Agent 协作模式：

| 模式 | 说明 |
|------|------|
| Supervisor-Worker | 监督者调度多个工作 Agent |
| Round-Robin | 轮询执行 |
| Parallel Execution | 并行执行 |
| Producer-Reviewer | 生产者-审核者 |
| Hierarchical Teams | 层级团队 |

多 Agent 可通过共享 Memory Block 实现知识共享。

## 6. 局限性

### 6.1 上下文窗口仍是硬约束

- Memory Blocks 本质上是在有限的上下文窗口中"切分"空间
- 核心记忆越多，留给实际对话的空间越少
- 前沿模型上限约 200k tokens

### 6.2 Token 开销

- 核心记忆**每次对话都注入上下文**，持续消耗 token
- 复杂的 Agent 循环（多步工具调用）累积延迟和成本
- Sleep-Time Compute 是异步的，但不适合实时场景

### 6.3 记忆质量依赖模型能力

- Agent 自主编辑记忆的质量取决于底层 LLM 的判断力
- 可能出现记忆幻觉、不一致、重要信息遗漏
- 记忆驱逐和摘要策略可能丢失关键细节

### 6.4 架构锁定

- **Agent 必须跑在 Letta 框架内**，用它的 Agent Loop
- 不能给已有的 Agent（如 Claude Code、Cursor）直接加记忆
- 与 LangChain、LlamaIndex 等框架的集成需要适配
- 自部署 Docker server 的运维复杂度高于简单 SDK 方案

### 6.5 V1 架构的权衡

- 非推理模型（如 GPT-4o mini）无法生成推理链
- 闭源 API 的推理 token 不可移植（跨模型无法传递推理状态）
- 废弃 heartbeat 后，自主循环需要自定义实现

## 7. 适用场景

| 场景 | 适配度 | 说明 |
|------|--------|------|
| 长期个人 AI 助手 | 高 | 需要深度记忆和持续学习 |
| 多轮对话系统 | 高 | 自动管理对话上下文 |
| 自主 Agent 应用 | 高 | Agent 需要自主决策和管理状态 |
| 游戏角色 NPC | 中 | 需要角色记忆和个性延续 |
| 开发工具记忆 | 低 | 架构锁定，无法直接集成到现有工具 |
| 轻量级记忆需求 | 低 | 引入整个平台过重 |

## 8. 商业模式

采用"**开源核心 + 托管服务**"模式：
- 核心框架 Apache 2.0 开源
- 提供 `api.letta.com` 云端托管 API 服务（按用量付费）
- Letta Code 等产品

## 9. 与本项目的关联

Letta 是一个完整的 Agent 运行时平台，其核心价值是让 Agent **自主管理自己的记忆**。但它的问题是：

1. **架构锁定**：Agent 必须跑在 Letta 内，无法给 Claude Code、Cursor、ChatGPT 等已有工具"外挂"记忆
2. **过重**：只为"存经验"引入整个 Agent 平台不划算
3. **记忆非结构化**：核心记忆是自由文本块，不是结构化的"问题-解决方案"经验

本项目需要的不是一个 Agent 平台，而是一个**轻量级的经验存储与检索 API**，任何 Agent 都能通过简单的 HTTP 调用来读写经验。
