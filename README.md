# GopherAI

一个基于 Go 的 AI 助手平台，支持多模型对话、SSE 流式回复、RAG 知识库检索、MCP 工具调用和文件管理。

## 功能特性

- 多模型对话：支持 OpenAI 兼容模型、本地模型与 RAG 场景
- 流式响应：SSE 实时输出，历史消息直接完整回放
- 思考模式：普通回答使用 `Think`，启用工具时使用 `ThoughtChain`
- 显式工具启用：仅本轮勾选的工具参与执行，不再隐式启用默认工具
- RAG 知识库：支持文档上传、切分、向量索引、按文件检索
- 文件管理：上传、下载、索引、删索引、删除文件
- MCP / TTS：支持 MCP 工具扩展与语音合成

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
docker-compose up -d

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

- `POST /api/v1/agent/*` 的 `tools` 表示“本次请求显式启用的工具列表”
- SSE 除 `meta` / `error` 外，`data` 直接发送完整的 `schema.Message` JSON
- 流结束后服务端追加 `data: [DONE]`
- 历史消息接口返回 `{ index, message, created_at }`，其中 `message` 为完整 `schema.Message`
- 历史消息读取遵循“Redis 优先，PostgreSQL 回源”，保证非流式生成后立即刷新也能读到最新 `reasoning_content`
- 后端通过领域 DAO 访问 Redis；service 只负责决定何时读写缓存与何时回源 PostgreSQL
- 首轮请求未携带 `session_id` 时，前端会在收到服务端返回的真实 `session_id` 后立即绑定当前会话，后续流式与非流式多轮对话都复用同一会话
- 非流式对话成功后，前端优先回查历史；若当前轮 assistant 尚未完成数据库异步落盘，则直接使用 `/agent/generate` 返回的 `message` 兜底展示

### SSE 示例

```text
data: {"type":"meta","session_id":"sess_001","message_index":4}
data: {"role":"assistant","reasoning_content":"先确认约束"}
data: {"role":"assistant","content":"答案","response_meta":{"finish_reason":"stop"}}
data: [DONE]
```

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
| GET | `/api/v1/sessions` | 会话列表 |
| DELETE | `/api/v1/sessions/:id` | 删除会话 |
| PUT | `/api/v1/sessions/:id/title` | 更新标题 |
| POST | `/api/v1/file/upload` | 上传文件 |
| GET | `/api/v1/file/list` | 文件列表 |
| GET | `/api/v1/file/url/:id` | 文件访问 URL |
| GET | `/api/v1/file/download/:id` | 下载文件 |
| DELETE | `/api/v1/file/:id` | 删除文件 |
| POST | `/api/v1/file/index/:id` | 创建索引 |
| DELETE | `/api/v1/file/index/:id` | 删除索引 |

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
