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
- `manager.go` 负责 Agent 创建、缓存和流式执行
- `tools/registry.go` 负责内置工具与 MCP 工具注册
- 请求中的 `tools` 仅代表本轮显式启用工具；未传或为空时不隐式启用默认工具
- Agent 缓存键包含 `sessionID + modelName + toolSignature`，避免不同工具组合复用同一实例

内置工具：
- `knowledge_search`
- `sequential_thinking`
- 动态加载的 MCP 工具

### 2. 流式消息链路

```text
用户消息 -> Agent 执行 -> SSE 输出 -> Redis 缓存 -> RabbitMQ 异步落 PostgreSQL
```

- `common/agent` 基于 `react.WithMessageFuture()` 逐条消费 `schema.Message` 流
- service 层负责会话创建、消息索引分配、持久化和最小 SSE 事件包装
- controller 层统一通过 Gin `c.Stream(...)` + `c.SSEvent("message", payload)` 输出
- SSE 仅保留 `meta` / `error` 控制包，其余 `data` 直接发送完整 `schema.Message`
- 流结束追加 `data: [DONE]`
- Redis 与 RabbitMQ 载荷都保留 `index`、`payload`、`tool_calls`
- 历史消息读取遵循“Redis 优先，PostgreSQL 回源”，以兼容“Redis 同步写、PostgreSQL 异步落库”的消息链路
- Redis 访问通过领域 DAO 收口；service 负责缓存策略，不直接调用 `common/redis` 的业务函数

消息模型关键字段：

```go
type Message struct {
    SessionID string
    Index     int
    Role      string
    Content   string
    Payload   json.RawMessage
    ToolCalls json.RawMessage
}
```

### 3. 前端聊天渲染约定

- 实时 SSE 消息使用 `renderMode=stream`，历史消息使用 `renderMode=instant`
- 未启用工具时，`reasoning_content` 使用 `Think` 展示
- 启用工具后，前端将 `assistant(tool_calls)`、`tool`、最终 `assistant` 重建为 `ThoughtChain`
- 工具目录通过 `GET /api/v1/tools` 动态拉取
- 非流式生成成功后优先回查历史；若本轮 assistant 尚未完成异步落库且未启用工具，前端使用 `/agent/generate` 返回的最终 `schema.Message` 做一次本地兜底，确保思考内容可立即显示

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
| GET | `/api/v1/sessions` | 会话列表 |
| DELETE | `/api/v1/sessions/:id` | 删除会话 |
| PUT | `/api/v1/sessions/:id/title` | 更新标题 |
| POST | `/api/v1/file/upload` | 上传文件 |
| GET | `/api/v1/file/list` | 文件列表 |
| GET | `/api/v1/file/url/:id` | 获取文件 URL |
| GET | `/api/v1/file/download/:id` | 下载文件 |
| DELETE | `/api/v1/file/:id` | 删除文件 |
| POST | `/api/v1/file/index/:id` | 创建索引 |
| DELETE | `/api/v1/file/index/:id` | 删除索引 |

### Agent 接口约定

- `tools` 字段是请求级显式工具列表，不写入会话配置
- SSE `data` 直接传 `schema.Message` JSON，不再拆分自定义 delta 事件
- 历史消息接口只允许读取当前用户自己的会话；前端直接读取 `response.data.data.messages`
- 会话列表接口只返回当前用户会话；前端直接读取 `response.data.data.sessions`

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

### 添加 MCP 工具

1. 在 `common/mcp/server/server.go` 注册工具
2. 在 Agent 工具注册表中暴露该工具

### 添加模型能力

1. 在 `common/llm/` 或相关模块补充实现
2. 在 Agent / service 调用链中接入
