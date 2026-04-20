# 为新 Agent 平台开发 Framework Renderer

## 概述

canned-exp 通过 `internal/renderer` 包支持将 Personality 数据渲染为不同 Agent 平台的配置文件。添加新平台只需实现接口 + 注册 + 写测试，无需改动其他层。

## 步骤

### 1. 创建 Renderer 文件

在 `internal/renderer/` 下新建文件，实现 `FrameworkRenderer` 接口：

```go
type FrameworkRenderer interface {
    Framework() string                                                    // 返回平台名，如 "openclaw"
    Render(agentName string, personalities map[string]string) []RenderedFile  // 生成配置文件
}
```

`RenderedFile` 结构：`{ConfigPath: "文件名", Content: "Markdown 内容"}`。

### 2. 实现 Render 方法

两种模式参考：

**多文件渲染器（参考 OpenClaw）：**
- 拆分为多个私有方法（`renderSOUL`、`renderIDENTITY` 等），每个方法返回一个 `RenderedFile`
- 用 `strings.Builder` 拼 Markdown
- 用 `map[key]` 读取人格属性，`if v, ok := p["key"]; ok` 检查存在性
- 即使属性为空也返回文件（保持结构完整）

**单文件渲染器（参考 Claude Code）：**
- 定义 `var orderedKeys []string` 控制字段顺序
- 遍历已知 key 再补充未知 key
- 返回单个 `RenderedFile`

### 3. 注册到 Registry

在 `internal/renderer/renderer.go` 的 `NewRegistry()` 中添加一行：

```go
r.Register(&MyFrameworkRenderer{})
```

Registry 是显式注册（非 init 自动注册），在 `NewRegistry()` 中集中管理所有渲染器。

**注意：** Registry 对未知 framework 有 fallback 行为（生成通用 CONFIG.md），所以新渲染器注册是可选的，但推荐实现平台专属渲染。

### 4. 写测试

在 `internal/renderer/my_framework_test.go` 中：

- 创建 `fullPersonality()` 辅助函数返回包含所有 16 个 key 的完整 map
- 直接构造 renderer 调用 `Render`，用 `strings.Contains` 断言内容
- 覆盖场景：完整属性、部分属性、空属性
- 多文件渲染器额外检查文件数量和每个文件的 ConfigPath

### 5. 无需其他改动

MCP Controller（`handleRenderPersonality`）通过 Registry 泛化调用，新增渲染器后自动生效。`render_personality` 工具的 `framework` 参数直接传入新平台名即可。

## 关键文件

| 文件 | 职责 |
|------|------|
| `internal/renderer/renderer.go` | 接口定义 + Registry |
| `internal/renderer/openclaw.go` | 多文件渲染器参考（6 文件） |
| `internal/renderer/claude_code.go` | 单文件渲染器参考 |
| `internal/mcp/controller/experience.go` | MCP 工具集成（无需改动） |

## 注意事项

- Personality key 由 `personality_keys` 表管理，新增平台不意味着新增 key——key 是平台无关的
- 渲染器只做格式转换，不存储数据
- 路由和 Bootstrap 层完全不需要改动
