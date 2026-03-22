# GopherAI

GopherAI 是一个前后端分离的 AI 助手平台，后端基于 Go + Gin，前端基于 React 19 + Ant Design 6，支持多模型对话、SSE 流式输出、DeepAgent、RAG 检索、文件管理和用户自定义 MCP。

## 核心能力

- 多模型对话：兼容 OpenAI 风格模型，并区分普通模型与思考模型
- 流式聊天：通过 SSE 输出增量消息，历史消息可直接回放
- 显式工具启用：每次请求只启用本轮勾选的内置工具和 MCP 服务
- DeepAgent：在聊天页切换模式即可使用独立运行时和用户专属工作区
- MCP 集成：支持用户配置远程 SSE / HTTP MCP 服务，并按服务整体启用
- RAG 检索：支持文件上传、索引、检索和向量召回
- 文件管理：支持上传、下载、创建索引、删除索引和删除文件

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.25 + Gin |
| 前端 | React 19 + React Router 7 + Ant Design 6 + Ant Design X |
| 数据库 | PostgreSQL + pgvector |
| 缓存 | Redis |
| 消息队列 | RabbitMQ |
| 对象存储 | MinIO |
| AI 框架 | CloudWeGo Eino |

## 快速开始

### 环境要求

- Go 1.25+
- Node.js LTS
- pnpm
- PostgreSQL（启用 `pgvector`）
- Redis
- RabbitMQ
- MinIO

### 启动步骤

```bash
# 1. 配置环境变量
cp .env.example .env

# 2. 启动依赖服务
docker compose up -d

# 3. 启用 DeepAgent 时，先构建运行镜像
docker build -t gopherai/deep-agent:latest -f docker/deepagent/Dockerfile .

# 4. 启动后端
go mod download
go run main.go

# 5. 启动前端
cd frontend
pnpm install
pnpm dev
```

默认地址：

- 前端：`http://localhost:5173`
- 后端：`http://localhost:9090`

## 关键约定

- `tools` 是请求级内置工具列表；未传或为空时，不隐式启用默认工具。
- `mcp_server_ids` 是请求级 MCP 服务 UUID 列表；启用粒度是“服务”，不是单个远程工具。
- DeepAgent 使用独立 `/api/v1/deep-agent/*` 接口，不和普通聊天接口混用模式字段。
- SSE 每帧统一输出 `{ type, code, message, response }`，事件类型固定为 `response.created`、`response.message.delta`、`response.message.completed`、`response.error`、`response.done`。
- 对外接口统一使用业务 UUID，数据库内部关联使用数值 `*_ref_id`。
- 历史消息读取遵循“Redis 优先，PostgreSQL 回源”。

更多开发约定、接口细节和实现边界见 [AGENTS.md](./AGENTS.md)。

## 目录概览

```text
GopherAI/
├── main.go
├── config/
├── controller/
├── service/
├── dao/
├── model/
├── common/
├── router/
├── middleware/
├── utils/
└── frontend/src/
```

## 配置说明

项目通过 `.env` 管理配置，完整字段见 `.env.example`。常用分类如下：

- 应用：`APP_NAME`、`APP_HOST`、`APP_PORT`
- 数据库：`POSTGRES_*`
- 缓存：`REDIS_*`
- 消息队列：`RABBITMQ_*`
- 鉴权：`JWT_*`
- 模型：`OPENAI_*`
- RAG：`RAG_*`
- 对象存储：`MINIO_*`
- MCP：`MCP_SECRET_KEY`
- DeepAgent：`DEEP_AGENT_*`

## 构建

```bash
# 后端
go build -o gopherai main.go

# 前端
cd frontend
pnpm build
```

## License

MIT License
