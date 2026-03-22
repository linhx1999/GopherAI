# GopherAI 项目上下文

## 项目概述

GopherAI 是一个前后端分离的 AI 助手平台，后端使用 Go + Gin，前端使用 React 19 + Ant Design 6，支持多模型对话、工具调用、RAG 检索、文件管理和 TTS。

### 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.25 + Gin |
| 前端 | React 19 + React Router 7 + Ant Design 6 + Ant Design X |
| 数据库 | PostgreSQL + pgvector |
| 缓存 | Redis |
| 消息队列 | RabbitMQ |
| 对象存储 | MinIO |
| AI 框架 | CloudWeGo Eino |
| 协议 | MCP |

## 重要开发规范

### 文档更新策略

每次代码变更后必须同步更新文档：
- 面向用户的变更更新 `README.md`
- 面向开发的变更更新 `AGENTS.md`
- 保持文档与代码实现一致

## 项目结构

```text
GopherAI/
├── main.go
├── .env / .env.example
├── config/
├── model/
├── dao/
├── service/
│   ├── agent/
│   ├── file/
│   ├── rag/
│   ├── session/
│   └── user/
├── controller/
│   ├── agent/
│   ├── file/
│   ├── session/
│   ├── tts/
│   └── user/
├── router/
├── middleware/jwt/
├── common/
│   ├── agent/
│   ├── rag/
│   ├── mcp/
│   ├── postgres/
│   ├── redis/
│   ├── rabbitmq/
│   ├── minio/
│   ├── llm/
│   ├── email/
│   └── tts/
├── utils/
└── frontend/src/
    ├── components/
    ├── hooks/
    ├── router/
    ├── utils/
    └── views/
        ├── Chat/
        ├── FileManager/
        ├── Login.jsx
        ├── Register.jsx
        └── Menu.jsx
```

## 核心架构

### 1. Agent 与工具系统

- 核心目录：`common/agent/`
- `manager.go` 负责基于全局 ChatModel 池按请求创建 Eino ADK `ChatModelAgent`
- `tools/registry.go` 仅维护全局工具 map；后端按请求中的工具名直接构造 `[]tool.BaseTool`
- 请求中的 `tools` 仅代表本轮显式启用的工具 API 名称；未传或为空时不隐式启用默认工具
- 内置工具标准调用名保持为 `knowledge_search` 和 `sequential_thinking`
- 请求中的未知工具名属于参数错误，必须在创建会话、写入消息和调用模型前被拦截
- `common/llm` 维护按模型名复用的全局 ChatModel 实例池；`thinking_mode` 通过选择不同全局模型实现

内置工具：
- `knowledge_search`
- `sequential_thinking`

### 2. 流式消息链路

```text
用户消息 -> Agent 执行 -> SSE 输出 -> Redis 缓存 -> RabbitMQ 异步落 PostgreSQL
```

- `common/agent` 基于 Eino ADK `ChatModelAgent` + `Runner` 消费 `AgentEvent`，并提取其中的 `schema.Message`
- service 层负责会话创建、消息索引分配、持久化和最小 SSE 事件包装
- controller 层统一通过 Gin `c.Stream(...)` + `c.SSEvent("message", payload)` 输出
- SSE 每一帧统一输出 `{ type, code, message, response }` envelope
- 流式事件固定使用 `response.created` / `response.message.delta` / `response.message.completed` / `response.error` / `response.done`
- Redis 与 RabbitMQ 载荷都保留 `index`、`payload`、`tool_calls`
- 历史消息读取遵循“Redis 优先，PostgreSQL 回源”，以兼容“Redis 同步写、PostgreSQL 异步落库”的消息链路
- Redis 访问通过领域 DAO 收口；service 负责缓存策略，不直接调用 `common/redis` 的业务函数
- 流式首轮请求改为先调用 `POST /api/v1/sessions` 获取真实会话，再调用 `POST /api/v1/agent/stream`
- 若 HTTP 请求上下文已取消，service/controller 应将其视为请求中断并停止写回，避免把客户端断开误记为模型失败

消息模型关键字段：

```go
type Message struct {
    MessageID    string
    SessionRefID uint
    UserRefID    uint
    Index        int
    Role         string
    Content      string
    Payload      json.RawMessage
    ToolCalls    json.RawMessage
}
```

### 模型标识约定

- `User`、`Session`、`File`、`DocumentChunk`、`Message` 全部使用 `gorm.Model` 管理数据库主键、时间字段和软删除
- 每个模型都保留独立业务 UUID：`user_id`、`session_id`、`file_id`、`chunk_id`、`message_id`
- 数据库内部关联统一使用数值字段：`user_ref_id`、`session_ref_id`、`file_ref_id`
- 项目不使用数据库外键约束；一致性由 service/dao 显式校验与删除顺序保证
- controller / service / 前端不能暴露数据库自增主键；公开接口只接收和返回业务 UUID
- `Session` 只保存会话元数据，不持久化工具列表；工具开关完全由请求级 `tools` 决定
- 旧标识结构不做兼容；启动时若检测到旧结构，应直接删除相关表并按新模型重建

### 3. 前端聊天渲染约定

- 实时 SSE 消息使用 `renderMode=stream`，历史消息使用 `renderMode=instant`
- 聊天页前端默认开启流式输出和思考模式；输入区开关只影响当前页面会话中的后续请求，不做本地持久化
- assistant 的思考和工具过程统一使用 `ThoughtChain` 展示；最终回答正文仍保留为独立 bubble
- 前端以 `response.message.completed` 作为 assistant/tool 消息的正式边界；`response.message.delta` 只负责增量渲染和 loading 态，不再单独决定 ThoughtChain 阶段切换
- assistant 的 `reasoning_content` 统一映射为 `ThoughtChain` 中的“深度思考”步骤；thinking mode 开启后，流式开始即可先显示 loading 的思考步骤
- assistant 完整消息若包含 `tool_calls`，思考内容保留为独立步骤，工具调用步骤的 `description` 仅承载调用前正文或简短说明，步骤 `title` 使用工具目录中的 API 名称映射展示名
- 当同一条 assistant 同时存在多段思考和工具步骤时，ThoughtChain 必须按步骤产生的时间顺序展示；典型顺序为“第一次深度思考 -> 工具调用 -> 工具结果 -> 第二次深度思考”
- ThoughtChain 统一通过 `defaultExpandedKeys` 默认展开全部步骤；reasoning 内容、工具调用参数和工具执行结果首次渲染时都应直接展示，用户可再手动折叠
- 前端识别“工具调用 assistant 消息”时，以 `response_meta.finish_reason=tool_calls` 为准；最终 assistant completed 即使仍附带历史 `tool_calls`，也必须继续渲染最终回答 bubble
- `role=tool` 的完整消息作为独立结果步骤挂到同一条 assistant 记录下；最终 assistant 完整消息才保留为正文 bubble
- 仅携带 metadata 的空 assistant delta 不应单独渲染成气泡
- 历史消息回放与流式过程共用同一套 ThoughtChain 聚合规则，刷新后应保持与流式阶段一致的步骤顺序和正文归属
- ThoughtChain 中的思考内容和 `role=tool` 结果内容统一使用 `XMarkdown` 渲染
- 聊天消息列表统一使用 `Bubble.List` 渲染；同一条 assistant 记录在前端展示阶段会拆成 `thought_chain` 列表项和最终回答列表项
- 列表滚动统一通过 `Bubble.ListRef.scrollTo` 控制；自动下滑只在用户当前接近底部且位于最后一页时生效
- 当 `Bubble.List` 内容在持续增长时，`scrollTo({ top: 'bottom', behavior: 'smooth' })` 可能被组件内部兼容逻辑退化为 `instant`，以保证列表继续贴底
- 工具目录通过 `GET /api/v1/tools` 动态拉取
- 工具目录中的 `name` 是 API 调用名，`display_name` 是前端展示名；前端不能把展示名回传给后端
- 点击“新建会话”仍只创建本地临时会话；首轮流式发送前前端必须先调用 `POST /api/v1/sessions`，拿到真实 `sessionId` 后再更新 `activeKey` 并发起 `/agent/stream`
- `/agent/stream` 的 SSE 不再下发 `message_index`；前端按 `response.*` 事件顺序驱动当前消息与 ThoughtChain 状态
- 非流式生成成功后优先回查历史；若本轮 assistant 尚未完成异步落库且未启用工具，前端使用 `/agent/generate` 返回的最终 `schema.Message` 做一次本地兜底，确保思考内容可立即显示
- 前端统一以服务端 `401` 或业务响应码 `2006` / `2007` 作为 token 失效判定；axios 响应拦截器与聊天页流式 `fetch` 都必须复用同一个未授权处理函数，清理本地 token 后跳转 `/login`

### 4. RAG 与文件流程

```text
上传文件 -> MinIO 存储 -> 手动触发索引 -> 文档切分/向量化 -> pgvector 检索 -> LLM 回答
```

- 关键目录：`common/rag/`、`service/rag/`、`common/minio/`
- 支持 Markdown 标题切分和固定长度切分
- 文件索引状态：`pending` / `indexing` / `indexed` / `failed`
- 删除文件时同时清理对象存储、数据库记录与向量索引

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
| GET | `/api/v1/agent/:session_id/messages` | 会话消息 |
| GET | `/api/v1/tools` | 工具列表 |
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

### Agent 接口约定

- `tools` 字段是请求级显式工具 API 名称列表，不写入会话配置
- 后端按请求中的工具名顺序去重后装配 `[]tool.BaseTool`；未知工具名直接返回参数错误
- `service/agent` 的生成与流式入口共享同一套显式参数准备逻辑，不再额外定义内部请求 DTO
- system prompt 通过 ADK `ChatModelAgentConfig.Instruction` 注入；会话消息数组只包含历史消息和当前用户消息
- SSE `data` 统一传 `{ type, code, message, response }`；流式消息内容位于 `response.delta` 或 `response.message`
- `/agent/stream` 必须携带已有 `session_id`；缺失时直接返回参数错误 SSE 事件
- 历史消息接口只允许读取当前用户自己的会话；前端直接读取 `response.data.data.messages`
- 会话列表接口只返回当前用户会话；前端直接读取 `response.data.data.sessions`
- 创建会话接口返回 `response.data.data.session`，结构与会话列表项一致：`sessionId` / `title` / `createdAt`
- 历史消息项新增 `message_id`；文件列表与文件操作统一使用 `file_id`

## 配置说明

配置通过 `.env` 管理，复制 `.env.example` 后填写实际值。

| 分类 | 变量 |
|------|------|
| 应用 | `APP_NAME` `APP_HOST` `APP_PORT` |
| PostgreSQL | `POSTGRES_HOST` `POSTGRES_PORT` `POSTGRES_USER` `POSTGRES_PASSWORD` `POSTGRES_DB` `POSTGRES_SSL_MODE` |
| Redis | `REDIS_HOST` `REDIS_PORT` `REDIS_PASSWORD` `REDIS_DB` |
| RabbitMQ | `RABBITMQ_HOST` `RABBITMQ_PORT` `RABBITMQ_USER` `RABBITMQ_PASSWORD` `RABBITMQ_VHOST` |
| JWT | `JWT_SECRET_KEY` `JWT_EXPIRE_DURATION` |
| Email | `EMAIL_AUTHCODE` `EMAIL_FROM` |
| OpenAI | `OPENAI_API_KEY` `OPENAI_MODEL_NAME` `OPENAI_REASONING_MODEL_NAME` `OPENAI_BASE_URL` |
| RAG | `RAG_EMBEDDING_MODEL` `RAG_CHAT_MODEL` `RAG_BASE_URL` `RAG_DOC_DIR` `RAG_DIMENSION` |
| TTS | `VOICE_API_KEY` `VOICE_SECRET_KEY` |
| MinIO | `MINIO_ENDPOINT` `MINIO_ACCESS_KEY` `MINIO_SECRET_KEY` `MINIO_BUCKET` `MINIO_USE_SSL` |

## 构建与运行

### 后端

```bash
cp .env.example .env
go mod download
go run main.go
go build -o gopherai main.go
```

### 前端

```bash
cd frontend
pnpm install
pnpm dev
pnpm build
pnpm lint
```

### 依赖服务

- PostgreSQL with pgvector
- Redis
- RabbitMQ
- MinIO

## 开发规范

### 分层约定

1. Controller：HTTP 请求和参数校验
2. Service：业务逻辑与缓存策略决策
3. DAO：数据访问，包含 PostgreSQL 与 Redis 的领域化读写

### 前端组件兼容性

- 项目使用 Ant Design 6，新增或修改前端组件时优先使用新属性名
- 例如 `Card` 使用 `variant` 代替 `bordered`，`Space` 使用 `orientation` 代替 `direction`
- 提交前应消除本项目源码中可直接修复的 Ant Design 弃用警告

### 响应结构

```go
type Response struct {
    Code int         `json:"code"`
    Msg  string      `json:"msg"`
    Data interface{} `json:"data"`
}
```

### JWT 认证

- 通过 `Authorization: Bearer <token>` 或 `?token=xxx` 传递
- 中间件解析用户名后写入 `gin.Context`

## 前端页面

| 路由 | 页面 |
|------|------|
| `/login` | `Login.jsx` |
| `/register` | `Register.jsx` |
| `/menu` | `Menu.jsx` |
| `/ai-chat` | `views/Chat/index.jsx` |
| `/file-manager` | `views/FileManager/index.jsx` |

聊天页重点：
- 支持普通响应和 SSE 流式响应
- 支持思考模式切换
- 支持动态工具目录、TTS、会话管理、RAG 文件选择

文件管理页重点：
- 顶部工具栏 + 主内容布局
- 逻辑集中在 `useFileManager`
- 支持上传、索引、删索引、删除文件和状态展示

## 常见开发任务

### 添加新的 API

1. 在 `model/` 定义模型
2. 在 `dao/` 添加数据访问
3. 在 `service/` 编写业务逻辑
4. 在 `controller/` 添加 Handler
5. 在 `router/` 注册路由

### 添加新的 Agent 工具

1. 在 `common/agent/tools/` 新增对应工具文件
2. 在全局工具 map 中注册该工具描述与构造函数

### 添加模型能力

1. 在 `common/llm/` 或相关模块补充实现
2. 在 Agent / service 调用链中接入
