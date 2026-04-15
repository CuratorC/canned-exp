# 竞品与相关项目调研

> 调研日期：2026-04-13
> 信息来源：[Awesome-Agent-Memory](https://github.com/TeleAI-UAGI/Awesome-Agent-Memory)、[Awesome-Memory-for-Agents](https://github.com/TsinghuaC3I/Awesome-Memory-for-Agents)

## 本项目定位

canned-exp 定位为：**轻量的、结构化的、API-first 的个人经验 CRUD 服务**，任何 AI Agent 都能通过 HTTP API 读写经验。

核心特征：
- 人或 Agent **主动写入**经验（非自动从对话提取）
- **结构化**存储（问题、解决方案、标签、上下文、验证状态）
- **跨平台、跨应用**（任何 Agent 通过 HTTP API 接入）
- 支持经验**迭代优化**（标记失效、补充方案、关联经验）

## 相关项目分类

### 第一梯队：与本项目高度相关

#### 1. Lorg

| 维度 | 详情 |
|------|------|
| 发布时间 | 2026-03 |
| 描述 | AI Agent 的永久性智能档案库。结构化贡献（prompts、workflows、insights、patterns）通过自动化质量关卡并 hash-chain 链接 |
| 存储 | 结构化存储 + hash-chain 不可篡改 |
| API | 有 |
| 许可证 | 待确认 |

**与本项目对比：**
- 相似：结构化经验、质量关卡、跨 Agent 使用
- 差异：面向 AI prompts/workflows 归档，非通用经验；有 hash-chain 机制（偏区块链思路）；不是通用经验 CRUD

#### 2. OMEGA

| 维度 | 详情 |
|------|------|
| 描述 | AI 编程 Agent 的持久化记忆，MCP server 提供 25 个工具 |
| 存储 | 待确认 |
| API | MCP 协议 |
| 亮点 | LongMemEval 基准测试第一名（95.4%） |

**与本项目对比：**
- 相似：面向编程 Agent、MCP 协议、结构化存储
- 差异：聚焦编程场景，不是通用经验库；绑死 MCP 协议

#### 3. SuperLocalMemory V2

| 维度 | 详情 |
|------|------|
| 发布时间 | 2026-02 |
| 描述 | 通用的本地优先记忆基础设施，支持 MCP + A2A 双协议 |
| 存储 | 本地存储 |
| API | MCP + A2A 协议 |

**与本项目对比：**
- 相似：本地优先、跨 Agent（MCP + A2A）、通用记忆层
- 差异：偏向运行时记忆管理，非结构化经验库；协议绑定较深

### 第二梯队：部分相关

#### 4. Memov

| 维度 | 详情 |
|------|------|
| 描述 | 基于 Git 的、可追溯的记忆层，专门为 Claude Code 设计 |
| 存储 | Git |
| API | Claude Code 插件 |

**与本项目对比：**
- 相似：给编码 Agent 用的记忆、可追溯
- 差异：绑死 Claude Code；用 Git 做存储而非数据库

#### 5. agentmemory

| 维度 | 详情 |
|------|------|
| 描述 | AI 编程 Agent 的持久化记忆 |
| 存储 | 待确认 |
| API | 待确认 |

**与本项目对比：**
- 相似：面向编程 Agent
- 差异：项目较早期，信息有限

#### 6. Hindsight

| 维度 | 详情 |
|------|------|
| 发布时间 | 2025-12 |
| 描述 | Agent 记忆系统，支持保留、召回和反思 |
| 存储 | 待确认 |
| 特点 | 有配套论文 |

**与本项目对比：**
- 相似：有"反思"机制，接近经验迭代
- 差异：学术项目，偏向对话记忆而非结构化经验

#### 7. ReMe（前 MemoryScope）

| 维度 | 详情 |
|------|------|
| 发布时间 | 2025-12 |
| 描述 | 动态程序性记忆框架，支持经验驱动的 Agent 进化 |
| 存储 | 待确认 |
| 特点 | 有配套论文 |

**与本项目对比：**
- 相似："经验驱动"的理念接近
- 差异：聚焦 Agent 自动提取经验，非人工主动写入

#### 8. Cognee

| 维度 | 详情 |
|------|------|
| 发布时间 | 2025-05 |
| 描述 | 优化知识图谱与 LLM 之间的接口，支持复杂推理 |
| 存储 | 知识图谱 |
| 许可证 | 开源 |

**与本项目对比：**
- 相似：结构化知识管理
- 差异：偏向知识图谱推理，非经验 CRUD；架构较重

### 第三梯队：值得了解但不直接相关

| 项目 | 特点 | 为什么不直接相关 |
|------|------|----------------|
| **Graphiti (Zep)** | 时序知识图谱 | 图数据库方案，偏重 |
| **MemOS** | 记忆操作系统 | 完整平台，过重 |
| **widemem-ai** | SQLite + FAISS，轻量级 | 技术栈类似但功能偏检索 |
| **MemU** | 通用记忆层 | 偏对话记忆 |
| **Congee** | 仿生记忆系统 | 学术研究导向 |
| **Second Me** | AI 数字分身 | 偏个人助理场景 |
| **MemMachine** | 记忆管理框架 | 偏学术 |
| **MemoryBear** | 记忆系统 | 偏对话记忆 |
| **Honcho** | Agent 上下文管理 | 偏短期记忆 |
| **LangMem** | LangChain 记忆模块 | 绑死 LangChain 生态 |
| **OpenMemory** | 开放记忆层 | 偏通用记忆，非经验 |
| **EverOS (EverMind)** | Agent 操作系统 | 完整平台 |
| **MIRIX** | 多 Agent 记忆系统 | 偏多 Agent 协作 |
| **Memobase** | 记忆评估基准 | 偏评估而非存储 |
| **TeleMem** | Mem0 的高性能替代 | 与 Mem0 同类 |
| **Mem9** | OpenClaw 记忆技能 | 绑死 OpenClaw |

## 关键学术资源

以下论文/综述对本项目设计有参考价值：

### 综述

- **Rethinking Memory in AI: Taxonomy, Operations, Topics, and Future Directions** (2025-05)
- **From Storage to Experience: A Survey on the Evolution of LLM Agent Memory Mechanisms** (2026)
- **Memory in the Age of AI Agents** (2025)

### 经验驱动方向

- **Learning from Experience**（清华论文集分类）
- **Remember Me, Refine Me: A Dynamic Procedural Memory Framework** (2025-12) — ReMe 论文
- **Hindsight is 20/20: Building Agent Memory that Retains, Recalls, and Reflects** (2025-12)
- **SWE-Exp: Experience-Driven Software Issue Resolution** (2025-07)
- **Agent KB: Leveraging Cross-Domain Experience for Agentic Problem Solving** (2025-07)
- **ExpeL: LLM Agents Are Experiential Learners** (2023-08)

## 结论

**当前没有完全匹配的开源项目。** 现有方案可分为三类：

1. **对话记忆类**（Mem0、Zep、LangMem 等）— 从对话自动提取，非结构化
2. **Agent 平台类**（Letta、MemOS 等）— 完整运行时，架构锁定
3. **编程 Agent 记忆类**（OMEGA、Memov、agentmemory 等）— 聚焦编码场景

canned-exp 的差异化在于：
- **结构化经验**而非自然语言记忆片段
- **API-first**，任何 Agent 都能接入，不绑定特定框架
- **轻量级**，SQLite 即可运行，不需要向量数据库或图数据库
- **经验生命周期管理**（创建、验证、迭代、归档、失效）
