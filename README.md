# GopherAI

一个基于 Go 的 AI 助手平台，支持多模型对话、SSE 流式回复、RAG 知识库检索和文件管理。

## 功能特性

- 多模型对话：支持 OpenAI 兼容模型、本地模型与 RAG 场景
- 流式响应：SSE 实时输出，历史消息直接完整回放
- 思考模式：聊天页默认开启思考模式，assistant 的思考和工具过程统一使用 `ThoughtChain` 展示；最终回答正文仍保留独立 bubble，流式按 `response.message.completed` 收口消息边界
- 显式工具启用：仅本轮勾选的工具参与执行，不再隐式启用默认工具
- RAG 知识库：支持文档上传、切分、向量索引、按文件检索
- 文件管理：上传、下载、索引、删索引、删除文件
- TTS：支持语音合成

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.25 + Gin |
| 前端 | React 19 + React Router 7 + Ant Design 6 + Ant Design X |
| 数据库 | PostgreSQL + pgvector |
| 缓存 | Redis |
| 消息队列 | RabbitMQ |
| 存储 | MinIO |
| AI 框架 | CloudWeGo Eino |

## 快速开始

### 环境要求

- Go 1.25+
- Node.js 16+
- PostgreSQL 16+（启用 `pgvector`）
- Redis 7+
- RabbitMQ 3.x
- MinIO

### 启动步骤

```bash
# 1. 配置环境变量
cp .env.example .env

# 2. 启动依赖服务
docker compose up -d

# 3. 启动后端
go mod download
go run main.go

# 4. 启动前端
cd frontend
pnpm install
pnpm dev
```

默认访问地址：
- 前端：`http://localhost:5173`
- 后端：`http://localhost:9090`

## 关键约定

- `POST /api/v1/agent/*` 的 `tools` 表示“本次请求显式启用的工具 API 名称列表”
- SSE 每一帧统一输出 `{ type, code, message, response }` envelope
- 流式事件固定使用 `response.created` / `response.message.delta` / `response.message.completed` / `response.error` / `response.done`
- 历史消息接口返回 `{ message_id, index, message, created_at }`，其中 `message` 为完整 `schema.Message`
- 历史消息读取遵循“Redis 优先，PostgreSQL 回源”，保证非流式生成后立即刷新也能读到最新 `reasoning_content`
- 后端通过领域 DAO 访问 Redis；service 只负责决定何时读写缓存与何时回源 PostgreSQL
- 持久化模型统一采用“`gorm.Model` + 业务 UUID”双标识；数据库内部关联走数值 ID，对外接口统一使用业务 UUID
- Session 模型不持久化工具列表；工具启用状态仅来自当前请求的 `tools`
- `GET /api/v1/tools` 同时返回 `name` 和 `display_name`：前者用于 API 调用，后者仅用于前端展示
- 后端工具按工具名拆分到 `common/agent/tools/*.go`；服务端仅保留一个全局工具 map，按请求中的工具名直接构造 `[]tool.BaseTool`
- 生成与流式执行共享同一套准备逻辑，服务端按请求参数直接装配 agent 与执行上下文，不再额外包装内部请求对象
- 内置工具标准调用名保持为 `knowledge_search` 和 `sequential_thinking`；前端应始终使用接口返回的 `name`，展示时使用 `display_name`
- `POST /api/v1/agent/*` 收到未知工具名时会直接返回请求参数错误，且不会创建会话、写入消息或触发模型调用
- 后端执行层基于 Eino ADK `ChatModelAgent` + `Runner`；底层 ChatModel 按模型名全局复用，ChatModelAgent 按请求创建
- 流式首轮对话改为“先 `POST /api/v1/sessions` 创建会话，再 `POST /api/v1/agent/stream` 拉取智能体输出”；`/agent/stream` 必须携带已有 `session_id`
- 流式 SSE 不再暴露 `message_index`；前端仅按事件顺序消费 `response.*` envelope
- 聊天页前端默认使用流式输出和思考模式；用户仍可通过输入区开关临时切换为非流式或关闭思考模式
- 非流式首轮对话仍可不传 `session_id`，由 `/agent/generate` 继续沿用现有隐式建会话逻辑
- 当客户端主动断开、页面刷新或请求上下文取消时，流式与非流式接口都会将其视为请求终止，不再记录为模型调用失败
- 非流式对话成功后，前端优先回查历史；若当前轮 assistant 尚未完成数据库异步落盘，则直接使用 `/agent/generate` 返回的 `message` 兜底展示
- 流式阶段中，`reasoning_content` 会先进入 `ThoughtChain` 的“深度思考”步骤；工具调用和工具结果继续作为后续步骤，`response.message.completed` 才正式提交各步骤和最终 assistant 正文
- assistant 同时带有 `reasoning_content` 与 `tool_calls` 时，思考内容会保留为独立的“深度思考”步骤，不再并入首个工具调用步骤说明
- 当同一条 assistant 同时包含多段思考与工具步骤时，ThoughtChain 严格按步骤产生的时间顺序展示；典型顺序为“第一次深度思考 -> 工具调用 -> 工具结果 -> 第二次深度思考”
- ThoughtChain 通过 `defaultExpandedKeys` 默认展开全部步骤；思考内容、工具参数和工具结果首次渲染时都会直接展示，用户仍可手动折叠
- 前端识别“工具调用 assistant 消息”时，以 `response_meta.finish_reason=tool_calls` 为准；即使最终 assistant completed 仍携带历史 `tool_calls`，也必须继续渲染最终回答 bubble
- 无工具但有 `reasoning_content` 的回答，也会显示单个 ThoughtChain 思考步骤；最终 assistant 完整消息仍保留为正文 bubble
- 历史消息与流式消息共用同一套 ThoughtChain 聚合规则，刷新后步骤顺序、描述和工具结果展示应与流式过程保持一致
- ThoughtChain 中的思考内容和工具结果内容都使用 `XMarkdown` 渲染，保持与正文 Markdown 一致的展示效果
- 聊天消息列表统一使用 `Bubble.List` 渲染；同一条 assistant 记录在前端会拆成 `ThoughtChain` 列表项和最终回答列表项
- 自动下滑仅在用户当前接近底部时生效；用户手动上翻查看历史后，不会被流式增量或 ThoughtChain 新步骤强行拉回底部
- 流式增长过程中通过 `Bubble.ListRef.scrollTo({ top: 'bottom', behavior: 'smooth' })` 跟随到底；当内容持续增长时，组件内部可能退化为 `instant` 以保证贴底
- 前端统一以服务端 `401` 或业务响应码 `2006` / `2007` 作为 token 失效判定；普通 axios 请求与流式 `fetch` 请求都会清理本地 token 并自动跳转 `/login`
- 前端基于 Ant Design 6 开发时，优先使用 `variant`、`orientation` 等新属性，避免继续使用 `bordered`、`direction` 这类已弃用 API

## 标识模型说明

- 数据库层：`users.id`、`sessions.id`、`files.id`、`document_chunks.id`、`messages.id` 都是 `gorm.Model` 提供的自增主键
- 业务层：`user_id`、`session_id`、`file_id`、`chunk_id`、`message_id` 都是 UUID；前端和 HTTP API 只使用这些业务标识
- 内部关联：`user_ref_id`、`session_ref_id`、`file_ref_id` 仅作为普通索引字段使用，不建立数据库外键约束
- `Session` 仅存储会话元数据，不保存工具开关；每轮可用工具由请求体 `tools` 决定
- 历史消息、文件管理、JWT 认证都基于业务 UUID 入参，再在后端解析到内部数值 ID
- 旧版标识结构不做数据兼容；服务启动时检测到旧结构会直接删除相关表并按新模型重建

### SSE 示例

```text
data: {"type":"response.created","code":1000,"message":"success","response":null}
data: {"type":"response.message.delta","code":1000,"message":"success","response":{"delta":{"role":"assistant","reasoning_content":"先确认约束"}}}
data: {"type":"response.message.delta","code":1000,"message":"success","response":{"delta":{"role":"assistant","content":"答案","response_meta":{"finish_reason":"stop"}}}}
data: {"type":"response.message.completed","code":1000,"message":"success","response":{"message":{"role":"assistant","content":"答案","reasoning_content":"先确认约束","response_meta":{"finish_reason":"stop"}}}}
data: {"type":"response.done","code":1000,"message":"success","response":null}
```

执行约定：
- system prompt 由 ADK `ChatModelAgentConfig.Instruction` 承载，不作为首条 system message 写入会话消息数组
- `thinking_mode` 继续保留，通过选择不同的全局 ChatModel 实例实现，而不是切换 Agent 缓存

## API 概览

### 公开接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/user/register` | 用户注册 |
| POST | `/api/v1/user/login` | 用户登录 |

### 认证接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/agent/generate` | 非流式对话 |
| POST | `/api/v1/agent/stream` | SSE 流式对话 |
| GET | `/api/v1/agent/:session_id/messages` | 会话消息 |
| GET | `/api/v1/tools` | 工具列表 |
| POST | `/api/v1/sessions` | 创建会话 |
| GET | `/api/v1/sessions` | 会话列表 |
| DELETE | `/api/v1/sessions/:session_id` | 删除会话 |
| PUT | `/api/v1/sessions/:session_id/title` | 更新标题 |
| POST | `/api/v1/file/upload` | 上传文件 |
| GET | `/api/v1/file/list` | 文件列表 |
| GET | `/api/v1/file/url/:file_id` | 文件访问 URL |
| GET | `/api/v1/file/download/:file_id` | 下载文件 |
| DELETE | `/api/v1/file/:file_id` | 删除文件 |
| POST | `/api/v1/file/index/:file_id` | 创建索引 |
| DELETE | `/api/v1/file/index/:file_id` | 删除索引 |

## 配置说明

配置通过 `.env` 管理，常用项如下：

| 分类 | 变量 |
|------|------|
| 应用 | `APP_NAME` `APP_HOST` `APP_PORT` |
| PostgreSQL | `POSTGRES_HOST` `POSTGRES_PORT` `POSTGRES_USER` `POSTGRES_PASSWORD` `POSTGRES_DB` `POSTGRES_SSL_MODE` |
| Redis | `REDIS_HOST` `REDIS_PORT` `REDIS_PASSWORD` `REDIS_DB` |
| RabbitMQ | `RABBITMQ_HOST` `RABBITMQ_PORT` `RABBITMQ_USER` `RABBITMQ_PASSWORD` `RABBITMQ_VHOST` |
| JWT | `JWT_SECRET_KEY` `JWT_EXPIRE_DURATION` |
| OpenAI | `OPENAI_API_KEY` `OPENAI_MODEL_NAME` `OPENAI_REASONING_MODEL_NAME` `OPENAI_BASE_URL` |
| RAG | `RAG_EMBEDDING_MODEL` `RAG_CHAT_MODEL` `RAG_BASE_URL` `RAG_DOC_DIR` `RAG_DIMENSION` |
| MinIO | `MINIO_ENDPOINT` `MINIO_ACCESS_KEY` `MINIO_SECRET_KEY` `MINIO_BUCKET` `MINIO_USE_SSL` |
| TTS | `VOICE_API_KEY` `VOICE_SECRET_KEY` |

## 项目结构

```text
GopherAI/
├── main.go
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
├── router/
├── common/
│   ├── agent/
│   ├── rag/
│   ├── mcp/
│   ├── postgres/
│   ├── redis/
│   ├── rabbitmq/
│   ├── minio/
│   ├── llm/
│   └── tts/
└── frontend/src/
    ├── components/
    ├── router/
    ├── utils/
    └── views/
        ├── Chat/
        ├── FileManager/
        ├── Login.jsx
        ├── Register.jsx
        └── Menu.jsx
```

## 部署

```bash
# 编译
go build -o gopherai main.go

# 运行
./gopherai
```

## License

MIT License
