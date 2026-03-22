# GopherAI 开发约定

## 项目概览

GopherAI 是一个前后端分离的 AI 助手平台，后端使用 Go + Gin，前端使用 React 19 + Ant Design 6，支持多模型对话、DeepAgent、工具调用、用户自定义 MCP、RAG 检索、文件管理和 TTS。

## 文档维护

- 代码行为变更后必须同步更新文档。
- 面向用户的变更更新 `README.md`。
- 面向开发的变更更新 `AGENTS.md`。
- 文档内容必须与当前实现保持一致。

## 关键目录

- `controller/`：HTTP 请求处理和入参校验
- `service/`：业务逻辑、缓存策略和跨模块编排
- `dao/`：PostgreSQL / Redis 领域化读写
- `common/agent/`：普通 Agent 执行入口
- `common/agent/tools/`：内置工具定义与解析
- `common/mcp/`：MCP 请求头加密、远程 client 和工具拉取
- `common/deepagent/`：DeepAgent 运行时、工作区和容器管理
- `common/llm/`：按模型名复用全局 ChatModel 实例池
- `common/rag/`、`service/rag/`：文档切分、向量化与检索
- `frontend/src/views/Chat/`：聊天页与 ThoughtChain 渲染

## 核心架构约定

### Agent / MCP / DeepAgent

- `common/agent/manager.go` 负责基于全局 ChatModel 池按请求创建 Eino ADK `ChatModelAgent`。
- `common/agent/tools/` 中除具体工具文件外，仅保留 `manager.go` 和 `interface.go` 作为收口文件。
- 内置工具标准调用名固定为 `knowledge_search` 和 `sequentialthinking`。
- 请求中的 `tools` 仅表示本轮显式启用的内置工具 API 名称；未传或为空时不启用默认工具。
- 请求中的未知工具名必须在创建会话、写入消息和调用模型前直接拦截为参数错误。
- 请求中的 `mcp_server_ids` 仅表示本轮显式启用的 MCP 服务业务 UUID；启用粒度是“服务”。
- 未知 MCP 服务 ID 或不属于当前用户的服务，必须在执行前直接拦截。
- `common/mcp/` 负责敏感请求头加密、按 `transport_type` 建立远程 SSE / streamable HTTP client，并在请求结束后清理 client。
- DeepAgent 使用独立 `/api/v1/deep-agent/*` 路由，请求体与普通聊天接口保持同构，不向 `/api/v1/agent/*` 增加模式字段。
- DeepAgent 额外注入 `write_todos`、`task`、`read_file`、`write_file`、`edit_file`、`glob`、`grep`、`execute` 等工具，且文件操作仅允许访问 `workspace/<userUUID>`。
- DeepAgent 工作区固定为仓库根目录下的 `workspace/<userUUID>`；首次使用为空目录，“重建工作区”会清空后重建。
- 同一用户的 DeepAgent 请求严格串行；运行中再次发起请求时直接返回 `CodeDeepAgentRuntimeBusy`。
- DeepAgent 工具调用失败不会直接终止 agent run；失败会作为 `role=tool` 结果返回给模型继续决策。只有请求取消、超时和 interrupt/rerun 等控制流错误会中断执行。
- `thinking_mode` 通过选择不同全局 ChatModel 实现，不通过缓存不同 Agent 实例实现。

### 流式消息链路

```text
用户消息 -> Agent 执行 -> SSE 输出 -> Redis 缓存 -> RabbitMQ 异步落 PostgreSQL
```

- `common/agent` 基于 Eino ADK `ChatModelAgent` + `Runner` 消费 `AgentEvent`，提取 `schema.Message`。
- service 层负责会话创建、消息索引分配、持久化和最小 SSE 事件包装。
- controller 层统一通过 Gin `c.Stream(...)` + `c.SSEvent("message", payload)` 输出。
- SSE 每帧统一输出 `{ type, code, message, response }`。
- 流式事件固定使用 `response.created`、`response.message.delta`、`response.message.completed`、`response.error`、`response.done`。
- Redis 与 RabbitMQ 载荷都保留 `index`、`payload`、`tool_calls`。
- 历史消息读取遵循“Redis 优先，PostgreSQL 回源”。
- Redis 访问通过 DAO 收口；service 负责缓存策略，不直接调用 `common/redis` 中的业务函数。
- 首轮流式请求需先调用 `POST /api/v1/sessions` 获取真实会话，再调用 `POST /api/v1/agent/stream`。
- `/api/v1/agent/stream` 必须携带已有 `session_id`。
- 请求上下文取消时，应视为请求中断并停止写回，不能误记为模型失败。

### 数据与标识

- `User`、`Session`、`File`、`DocumentChunk`、`Message` 都使用 `gorm.Model` 管理数据库主键、时间字段和软删除。
- 每个模型都保留独立业务 UUID：`user_id`、`session_id`、`file_id`、`chunk_id`、`message_id`。
- 数据库内部关联统一使用数值字段：`user_ref_id`、`session_ref_id`、`file_ref_id`。
- 项目不使用数据库外键约束；一致性由 service / dao 显式校验与删除顺序保证。
- controller、service 和前端都不能暴露数据库自增主键；公开接口只接收和返回业务 UUID。
- `Session` 只保存会话元数据，不持久化工具列表；工具开关完全由请求级 `tools` / `mcp_server_ids` 决定。
- `MCPServer` 使用 `gorm.Model` + `mcp_server_id`，敏感请求头通过 `MCP_SECRET_KEY` 加密后保存在 `headers_ciphertext`。
- `DeepAgentRuntime` 使用 `gorm.Model` + `deep_agent_runtime_id`，按 `user_ref_id` 唯一绑定一个运行时记录。
- 旧标识结构不做兼容；启动时若检测到旧结构，应删除相关表并按新模型重建。

### 前端聊天不变量

- 实时 SSE 消息使用 `renderMode=stream`，历史消息使用 `renderMode=instant`。
- 聊天页默认开启流式输出和思考模式；输入区开关只影响当前页面会话中的后续请求。
- assistant 的思考和工具过程统一使用 `ThoughtChain` 展示，最终回答正文保留独立 bubble。
- 前端必须把 `response.message.delta` 和 `response.message.completed` 作为两类事件处理：`delta` 只更新活动缓冲，`completed` 作为最终真值覆盖对应内容。
- `reasoning_content` 统一映射为 ThoughtChain 中的思考步骤。
- `role=tool` 的完整消息作为工具结果步骤挂到同一条 assistant 记录下；最终 assistant 完整消息才保留为正文 bubble。
- 识别“工具调用 assistant 消息”时，以 `response_meta.finish_reason=tool_calls` 为准。
- 仅携带 metadata 的空 assistant delta 不应单独渲染成气泡。
- 历史消息回放与流式过程共用同一套 ThoughtChain 聚合规则。
- ThoughtChain 中的思考内容、工具调用参数和工具结果统一使用 `XMarkdown` 渲染。
- 工具目录通过 `GET /api/v1/tools` 动态拉取；返回内置工具列表、当前用户 MCP 服务摘要和 `deep_agent_enabled`。
- 聊天页切到 DeepAgent 时，发送目标改为 `/api/v1/deep-agent/generate` 或 `/api/v1/deep-agent/stream`。

### RAG 与文件

```text
上传文件 -> MinIO 存储 -> 手动触发索引 -> 文档切分/向量化 -> pgvector 检索 -> LLM 回答
```

- 支持 Markdown 标题切分和固定长度切分。
- 文件索引状态固定为 `pending`、`indexing`、`indexed`、`failed`。
- 删除文件时要同时清理对象存储、数据库记录与向量索引。

## API 路由

### 公开路由

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/user/register` | 用户注册 |
| POST | `/api/v1/user/login` | 用户登录 |

### 认证路由

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/agent/generate` | 非流式生成 |
| POST | `/api/v1/agent/stream` | SSE 流式生成 |
| POST | `/api/v1/deep-agent/generate` | DeepAgent 非流式生成 |
| POST | `/api/v1/deep-agent/stream` | DeepAgent SSE 流式生成 |
| GET | `/api/v1/deep-agent/runtime` | DeepAgent 运行时状态 |
| POST | `/api/v1/deep-agent/runtime/restart` | 重启 DeepAgent 容器 |
| POST | `/api/v1/deep-agent/runtime/rebuild` | 重建 DeepAgent 工作区与容器 |
| GET | `/api/v1/agent/:session_id/messages` | 会话消息 |
| GET | `/api/v1/tools` | 工具列表 |
| GET | `/api/v1/mcp/servers` | MCP 服务列表 |
| GET | `/api/v1/mcp/servers/:server_id` | MCP 服务详情 |
| POST | `/api/v1/mcp/servers` | 创建 MCP 服务 |
| PUT | `/api/v1/mcp/servers/:server_id` | 更新 MCP 服务 |
| DELETE | `/api/v1/mcp/servers/:server_id` | 删除 MCP 服务 |
| POST | `/api/v1/mcp/servers/:server_id/test` | 测试 MCP 连接并刷新工具快照 |
| POST | `/api/v1/sessions` | 创建会话 |
| GET | `/api/v1/sessions` | 会话列表 |
| DELETE | `/api/v1/sessions/:session_id` | 删除会话 |
| PUT | `/api/v1/sessions/:session_id/title` | 更新标题 |
| POST | `/api/v1/file/upload` | 上传文件 |
| GET | `/api/v1/file/list` | 文件列表 |
| GET | `/api/v1/file/url/:file_id` | 获取文件 URL |
| GET | `/api/v1/file/download/:file_id` | 下载文件 |
| DELETE | `/api/v1/file/:file_id` | 删除文件 |
| POST | `/api/v1/file/index/:file_id` | 创建索引 |
| DELETE | `/api/v1/file/index/:file_id` | 删除索引 |

## 开发规范

### 分层约定

1. Controller：HTTP 请求和参数校验
2. Service：业务逻辑与缓存策略决策
3. DAO：数据访问，包含 PostgreSQL 与 Redis 的领域化读写

### 前端约定

- 项目使用 Ant Design 6，新增或修改组件时优先使用新属性名。
- 例如 `Card` 使用 `variant` 代替 `bordered`，`Space` 使用 `orientation` 代替 `direction`。
- 提交前应消除本项目源码中可直接修复的 Ant Design 弃用警告。

### 响应结构

```go
type Response struct {
    Code int         `json:"code"`
    Msg  string      `json:"msg"`
    Data interface{} `json:"data"`
}
```

### JWT 认证

- 通过 `Authorization: Bearer <token>` 或 `?token=xxx` 传递。
- 中间件解析用户名后写入 `gin.Context`。

## 常见开发任务

### 添加新的 API

1. 在 `model/` 定义模型
2. 在 `dao/` 添加数据访问
3. 在 `service/` 编写业务逻辑
4. 在 `controller/` 添加 Handler
5. 在 `router/` 注册路由

### 添加新的 Agent 工具

1. 在 `common/agent/tools/` 新增对应工具文件
2. 在工具文件或工具目录定义中补充展示元信息，并在 `manager.go` 中接入实例构造逻辑

### 添加模型能力

1. 在 `common/llm/` 或相关模块补充实现
2. 在 Agent / service 调用链中接入
