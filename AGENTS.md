# GopherAI 项目上下文

## 项目概述

GopherAI 是一个基于 Go 语言开发的 AI 助手平台，提供多模型 AI 对话、图像识别、语音合成、RAG 知识库检索等功能。项目采用前后端分离架构，后端使用 Gin 框架，前端使用 React 19 + Ant Design 6。

### 技术栈

| 层级 | 技术 |
|------|------|
| 后端框架 | Go 1.25 + Gin |
| 前端框架 | React 19 + React Router 7 + Ant Design 6 + Ant Design X |
| 构建工具 | Vite 7 |
| ORM | GORM |
| 数据库 | PostgreSQL + pgvector |
| 缓存 | Redis |
| 消息队列 | RabbitMQ |
| 对象存储 | MinIO |
| AI 框架 | CloudWeGo Eino |
| 协议 | MCP (Model Context Protocol) |

---

## 重要开发规范

### 文档更新策略

**关键：每次代码变更后必须更新 README.md 和 AGENTS.md**

进行代码修改时，必须同步更新相关文档：
- 面向用户的变更（功能、安装、使用说明）需更新 `README.md`
- 面向开发的变更（架构、命令、工作流、内部系统）需更新 `AGENTS.md`
- 始终保持文档与代码同步
- 确保所有文档准确且及时

---

## 项目结构

```
GopherAI/
├── main.go                 # 入口文件
├── .env                    # 环境变量配置文件
├── .env.example            # 配置模板
├── config/                 # 配置管理
│   └── config.go           # 配置加载逻辑
├── model/                  # 数据模型定义
│   ├── user.go             # 用户模型
│   ├── session.go          # 会话模型（含 tools 字段）
│   ├── message.go          # 消息模型（含 index、payload、tool_calls 字段）
│   ├── file.go             # 文件模型
│   └── document_chunk.go   # 文档分块模型（RAG 向量存储）
├── dao/                    # 数据访问层
│   ├── user/
│   ├── session/
│   ├── message/
│   └── file/
├── service/                # 业务逻辑层
│   ├── user/
│   ├── session/
│   ├── image/
│   ├── file/
│   └── rag/                # RAG 索引服务
├── controller/             # 控制器层 (HTTP Handler)
│   ├── common.go           # 统一响应结构
│   ├── user/
│   ├── session/
│   ├── image/
│   ├── file/
│   └── tts/
├── router/                 # 路由注册
│   ├── router.go           # 路由初始化
│   ├── agent.go            # Agent 相关路由（generate / stream）
│   ├── Image.go            # 图像相关路由
│   ├── File.go             # 文件相关路由
│   └── user.go             # 用户相关路由
├── middleware/             # 中间件
│   └── jwt/jwt.go          # JWT 认证中间件
├── common/                 # 公共组件
│   ├── aihelper/           # AI 助手核心模块
│   │   ├── aihelper.go     # AIHelper 实体
│   │   ├── manager.go      # AIHelperManager 管理器
│   │   ├── model.go        # 多模型实现 (OpenAI/Ollama/RAG/MCP)
│   │   └── factory.go      # 模型工厂
│   ├── rag/                # RAG 检索增强生成
│   ├── mcp/                # MCP 服务 (天气查询示例)
│   ├── minio/              # MinIO 对象存储
│   │   └── minio.go        # MinIO 客户端初始化和操作封装
│   ├── postgres/           # PostgreSQL 初始化
│   │   └── postgres.go     # PostgreSQL 连接、迁移、向量索引
│   ├── redis/              # Redis 初始化（仅用于验证码缓存）
│   ├── rabbitmq/           # RabbitMQ 消息队列
│   ├── image/              # 图像识别
│   ├── email/              # 邮件服务
│   └── tts/                # 语音合成服务
├── utils/                  # 工具函数
│   ├── utils.go
│   └── myjwt/jwt.go        # JWT 工具
└── frontend/               # React 前端项目
    └── src/
        ├── components/     # 公共组件
        ├── hooks/          # 自定义 Hooks
        ├── views/          # 页面组件
        │   ├── AIChat/     # AI 对话页面
        │   │   ├── index.jsx
        │   │   ├── index.css
        │   │   └── config/constants.js
        │   ├── FileManager/# 文件管理页面
        │   │   ├── index.jsx
        │   │   └── index.css
        │   ├── Login.jsx
        │   ├── Register.jsx
        │   ├── Menu.jsx
        │   └── ImageRecognition.jsx
        ├── router/         # 前端路由
        └── utils/api.js    # API 封装
```

---

## 核心架构

### 1. Agent Manager 模块 (核心)

Agent Manager 是项目的核心组件，基于 CloudWeGo Eino 的 React Agent 实现：

```
用户请求 → 选择工具 → Agent Manager 创建/获取 Agent → React Agent 执行工具调用 → 生成响应
```

**关键文件**: `common/agent/`

| 文件 | 职责 |
|------|------|
| `manager.go` | AgentManager，管理 Agent 实例、动态工具加载、流式响应 |
| `tools/registry.go` | 工具注册表，管理内置工具和 MCP 工具 |
| `tools/rag_tool.go` | RAG 知识库检索工具 |
| `tools/mcp_tool.go` | MCP 工具封装（使用官方 eino-ext 实现） |

**动态工具加载**:
- 根据请求中的 `tools` 参数动态创建 Agent；`tools` 仅表示本次请求显式启用的工具，未传或传空时不会再隐式启用默认工具
- 支持 `knowledge_search`、`sequential_thinking`、MCP 工具
- 前端工具下拉通过 `GET /api/v1/tools` 动态拉取，后端返回 `name`、`display_name`、`description`、`category`
- Agent 缓存键包含 `sessionID + modelName + toolSignature`，避免不同工具组合复用同一实例

**流式读取约定**:
- Agent 流式执行使用 `react.WithMessageFuture()` 捕获 Agent 过程中的每一条产出消息流
- `common/agent` 逐条消费 `MessageFuture.GetMessageStreams()` 返回的 `*schema.StreamReader[*schema.Message]`，原样向下游转发 `schema.Message` chunk，并在每条产出消息结束后用 `schema.ConcatMessages()` 聚合完整消息
- 服务层负责会话创建、消息索引分配、Redis/RabbitMQ 持久化，以及把控制包和 `schema.Message` 包装为最小 SSE 事件流
- Controller 层由 `StreamHandler` 直接消费 service 返回的事件流，并统一调用 Gin `c.Stream(...)` + `c.SSEvent("message", payload)` 输出 SSE；channel 关闭时追加 `data: [DONE]`，客户端断连通过 `request.Context()` 继续向 service/Agent 链路传递取消信号
- SSE 只保留首包 `meta` 和错误包；其余 `data` 直接发送完整的 `schema.Message` JSON
- 前端按当前活跃产出消息累积 chunk，当 chunk 的 `response_meta.finish_reason` 非空时结束当前消息；若底层缺失该字段，后端会补一个仅含 `finish_reason=stop` 的终止 chunk
- 前端本地消息记录会额外维护仅用于展示的 `renderMode`：实时 SSE 产出的当前消息使用 `stream` 逐字显示，历史消息接口返回的记录统一使用 `instant` 直接完整渲染，避免切换会话或刷新后重复打字回放
- 前端消息记录会额外维护展示元数据：`renderMode` 用于区分实时逐字渲染和历史瞬时渲染，`expectReasoning` 用于驱动 `Think` 的 loading，占用工具链路时还会标记 `assistantRenderMode=tool_chain` 和本轮 `enabledToolNames`
- 当前轮次未启用工具时，assistant 的 `reasoning_content` 使用 `Think` 展示；一旦本轮启用工具，前端会把 `assistant(tool_calls)`、`tool`、最终 `assistant` 重建为 `ThoughtChain`，不再同时展示 `Think`

### 2. 消息存储流程

```
用户消息 → Agent 执行 → SSE 流式响应 → 保存消息到 Redis 缓存 → RabbitMQ 异步写入 PostgreSQL
```

消息通过 RabbitMQ 异步存储，避免阻塞主流程。
- Redis 缓存与 RabbitMQ 载荷都会保留 `index`、`payload` 和 `tool_calls`，保证历史回放与前端展示使用同一份 `schema.Message`

**消息模型**:
```go
type Message struct {
    ID        uint            // 主键
    SessionID string          // 会话 ID
    Index     int             // 线性索引（用于排序与回放）
    Role      string          // user/assistant/tool
    Content   string          // 消息内容
    Payload   json.RawMessage // 完整 schema.Message JSON
    ToolCalls json.RawMessage // 工具调用记录
    CreatedAt time.Time
}
```

### 3. RAG 流程

```
用户上传文档 → MinIO 存储 → 手动触发索引 → 文档切分 → 向量化 → PostgreSQL pgvector 存储 → 用户选择文件 → 向量检索 → 构建增强 Prompt → LLM 回答
```

**关键文件**: 
- `common/rag/rag.go` - RAG 索引和检索核心实现
- `common/postgres/postgres.go` - PostgreSQL + pgvector 初始化
- `model/document_chunk.go` - 文档分块向量模型
- `service/rag/rag.go` - 文件索引服务

**新特性**:
- 支持文档切分（Markdown 标题切分、文本固定长度切分）
- 支持指定文件 ID 进行 RAG 检索
- 文件索引状态管理（pending/indexing/indexed/failed）
- 独立的文件管理页面
- 使用 pgvector 进行向量存储和检索（IVFFlat 索引，余弦相似度）

### 4. 工具系统

**内置工具**:
- `knowledge_search` - RAG 知识库检索
- `sequential_thinking` - 逐步思考（使用官方 eino-ext 实现）

**MCP 工具**:
- 支持连接外部 MCP 服务器
- 使用 `eino-ext/components/tool/mcp` 封装
- 动态加载工具列表

**工具注册表** (`common/agent/tools/registry.go`):
```go
// 解析工具实例
tools := registry.ResolveTools(ctx, []string{"knowledge_search", "sequential_thinking"}, fileIDs)

// 列出所有可用工具
toolList := registry.ListAvailableTools(ctx)
```

### 5. MCP 服务

```
MCP Server → 注册工具 → HTTP SSE 接口 → MCP Client 连接 → 工具调用
```

**示例工具**: 天气查询 (`common/mcp/server/server.go`)

### 6. 文件存储流程

```
用户上传文件 → MinIO 对象存储 → 数据库记录元数据 → RAG 向量索引
```
- 下载：`DownloadFile()` 从 MinIO 获取文件内容
- 删除：`DeleteFile()` 删除 MinIO 对象、数据库记录、向量索引
- 列表：`GetFileList()` 查询用户所有文件

---

## API 路由

### 公开路由 (无需认证)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/user/register` | 用户注册 |
| POST | `/api/v1/user/login` | 用户登录 |

### 认证路由 (需要 JWT)

**Agent 对话**:
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/agent/generate` | 非流式生成消息 |
| POST | `/api/v1/agent/stream` | SSE 流式生成消息 |
| GET | `/api/v1/agent/:session_id/messages` | 获取当前用户的会话消息列表 |

**工具接口**:
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/tools` | 获取可用工具列表 |

**会话管理**:
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/sessions` | 获取当前用户会话列表 |
| DELETE | `/api/v1/sessions/:id` | 删除会话 |
| PUT | `/api/v1/sessions/:id/title` | 更新会话标题 |

**图像识别**:
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/image/recognize` | 图像识别 |

**文件管理**:
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/file/upload` | 上传文件 |
| GET | `/api/v1/file/list` | 获取文件列表 |
| GET | `/api/v1/file/url/:id` | 获取文件访问 URL |
| GET | `/api/v1/file/download/:id` | 下载文件 |
| DELETE | `/api/v1/file/:id` | 删除文件 |
| POST | `/api/v1/file/index/:id` | 手动触发文件索引 |
| DELETE | `/api/v1/file/index/:id` | 删除文件索引 |

### Agent 接口详细说明

#### POST /api/v1/agent/generate

**请求体**:
```json
{
  "session_id": "sess_001",           // [可选] 缺省时自动创建新会话
  "message": "帮我查一下天气",         // [必填] 用户新消息
  "tools": ["knowledge_search", "sequential_thinking"], // [可选] 工具配置
  "thinking_mode": true               // [可选] 是否启用思考模型
}
```

**响应体**:
```json
{
  "code": 1000,
  "msg": "success",
  "data": {
    "session_id": "sess_001",
    "message_index": 4,
    "message": {
      "role": "assistant",
      "reasoning_content": "先确认需求边界...",
      "content": "北京今天 25 度...",
      "tool_calls": [...]
    }
  }
}
```

#### POST /api/v1/agent/stream

**请求体**:
```json
{
  "session_id": "sess_001",           // [可选] 缺省时自动创建新会话
  "message": "帮我查一下天气",         // [必填] 用户新消息
  "tools": ["knowledge_search", "sequential_thinking"], // [可选] 工具配置
  "thinking_mode": true               // [可选] 是否启用思考模型
}
```

**SSE 流式事件格式**:
```
data: {"type": "meta", "session_id": "sess_001", "message_index": 4}
data: {"role":"assistant","reasoning_content":"先分析需求"}
data: {"role":"assistant","content":"北"}
data: {"role":"assistant","content":"京","response_meta":{"finish_reason":"stop"}}
data: {"role":"tool","tool_name":"knowledge_search","content":"...","response_meta":{"finish_reason":"stop"}}
```

说明：
- 除 `meta`/`error` 外，SSE `data` 直接是 `schema.Message` JSON，不再发送 `reasoning_delta`、`content_delta`、`message_end` 等自定义事件
- HTTP SSE 由 Controller 直接消费 service 事件流并统一使用 Gin `c.Stream` 逐条输出；channel 关闭时追加 `data: [DONE]`，客户端断连时取消信号会沿 `request.Context()` 继续传递给上游流式生成
- 前端同时展示 `reasoning_content` 和 `content`，并按 `response_meta.finish_reason` 切分多条产出消息
- 中间的 `assistant(tool_calls)` 与 `tool` 消息会在前端重建为 `ThoughtChain` 链路节点，最终 `assistant` 文本消息仍作为主气泡展示
- `POST /api/v1/agent/*` 的 `tools` 字段表示“本次请求显式启用的工具列表”，请求级传递，不写入会话配置

#### GET /api/v1/agent/:session_id/messages

**响应体**:
```json
{
  "code": 1000,
  "msg": "success",
  "data": {
    "session_id": "sess_001",
    "messages": [
      {
        "index": 0,
        "message": {
          "role": "user",
          "content": "你好"
        },
        "created_at": "2026-03-08T23:53:44+08:00"
      }
    ],
    "total": 1
  }
}
```

说明：
- 仅允许读取当前用户自己的会话历史；`session_id` 不存在或不属于当前用户时统一返回 `CodeSessionNotFound`
- service 层会先校验会话归属，再访问 Redis/DB 历史消息，避免仅凭 `session_id` 读取他人消息
- 前端聊天页直接读取 `response.data.data.messages`

#### GET /api/v1/sessions

**响应体**:
```json
{
  "code": 1000,
  "msg": "success",
  "data": {
    "sessions": [
      {
        "sessionId": "sess_001",
        "title": "新会话",
        "createdAt": "2026-03-08T23:53:44+08:00"
      }
    ]
  }
}
```

说明：
- 只返回当前用户自己的会话
- 前端聊天页直接读取 `response.data.data.sessions`，并按 `createdAt` 排序

### 可用工具

| 工具名称 | 说明 |
|----------|------|
| `knowledge_search` | 从知识库检索相关文档 |
| `sequential_thinking` | 逐步分析和解决复杂问题 |
| MCP 工具 | 通过 MCP 服务器动态加载的工具 |

---

## 配置说明

配置通过 `.env` 环境变量文件管理。复制 `.env.example` 为 `.env` 并填写实际值。

### 配置项列表

| 分类 | 环境变量 | 说明 |
|------|----------|------|
| **应用** | `APP_NAME` | 应用名称 |
| | `APP_HOST` | 监听地址 |
| | `APP_PORT` | 服务端口 |
| **PostgreSQL** | `POSTGRES_HOST` | PostgreSQL 主机 |
| | `POSTGRES_PORT` | PostgreSQL 端口 |
| | `POSTGRES_USER` | 用户名 |
| | `POSTGRES_PASSWORD` | 密码 |
| | `POSTGRES_DB` | 数据库名 |
| | `POSTGRES_SSL_MODE` | SSL 模式 |
| **Redis** | `REDIS_HOST` | Redis 主机 |
| | `REDIS_PORT` | Redis 端口 |
| | `REDIS_PASSWORD` | Redis 密码 |
| | `REDIS_DB` | 数据库索引 |
| **RabbitMQ** | `RABBITMQ_HOST` | RabbitMQ 主机 |
| | `RABBITMQ_PORT` | RabbitMQ 端口 |
| | `RABBITMQ_USER` | 用户名 |
| | `RABBITMQ_PASSWORD` | 密码 |
| | `RABBITMQ_VHOST` | 虚拟主机 |
| **JWT** | `JWT_SECRET_KEY` | JWT 密钥 |
| | `JWT_EXPIRE_DURATION` | 过期时间(小时) |
| **Email** | `EMAIL_AUTHCODE` | 邮箱授权码 |
| | `EMAIL_FROM` | 发件人邮箱 |
| **OpenAI** | `OPENAI_API_KEY` | API 密钥 |
| | `OPENAI_MODEL_NAME` | 默认对话模型名称 |
| | `OPENAI_REASONING_MODEL_NAME` | 思考模式模型名称 |
| | `OPENAI_BASE_URL` | API 基础 URL |
| **RAG** | `RAG_EMBEDDING_MODEL` | 嵌入模型 |
| | `RAG_CHAT_MODEL` | 对话模型 |
| | `RAG_BASE_URL` | 阿里云 API URL |
| | `RAG_DOC_DIR` | 文档目录 |
| | `RAG_DIMENSION` | 向量维度 |
| **语音** | `VOICE_API_KEY` | 百度语音 API Key |
| | `VOICE_SECRET_KEY` | 百度语音 Secret Key |
| **MinIO** | `MINIO_ENDPOINT` | MinIO 服务地址 |
| | `MINIO_ACCESS_KEY` | 访问密钥 |
| | `MINIO_SECRET_KEY` | 秘密密钥 |
| | `MINIO_BUCKET` | 存储桶名称 |
| | `MINIO_USE_SSL` | 是否使用 SSL |

---

## 构建与运行

### 后端

```bash
# 1. 复制配置文件
cp .env.example .env
# 编辑 .env 填写实际配置值

# 2. 安装依赖
go mod download

# 3. 运行服务
go run main.go

# 4. 编译
go build -o gopherai main.go
```

### 前端

```bash
cd frontend

# 安装依赖
pnpm install

# 开发模式
pnpm dev

# 生产构建
pnpm build

# 代码检查
pnpm lint
```

### 依赖服务

启动前需确保以下服务可用：
- PostgreSQL with pgvector (端口 5432)
- Redis (端口 6379)
- RabbitMQ (端口 5672)
- MinIO (端口 9000 API, 9001 控制台)

### Docker Compose

```bash
# 启动所有依赖服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 停止服务
docker-compose down
```

---

## 开发规范

### 代码分层

项目遵循经典三层架构：
1. **Controller**: 处理 HTTP 请求，参数校验
2. **Service**: 业务逻辑
3. **DAO**: 数据库操作

### 响应结构

```go
type Response struct {
    Code int         `json:"code"`
    Msg  string      `json:"msg"`
    Data interface{} `json:"data"`
}
```

### 错误码

定义在 `common/code/code.go`，使用统一的错误码规范。

### JWT 认证

- Token 通过 `Authorization: Bearer <token>` 或 URL 参数 `?token=xxx` 传递
- 中间件从 Token 中解析用户名并存入 `gin.Context`

---

## 前端页面

| 路由 | 页面 | 说明 |
|------|------|------|
| `/login` | Login.jsx | 登录页 |
| `/register` | Register.jsx | 注册页 |
| `/menu` | Menu.jsx | 主菜单 |
| `/ai-chat` | AIChat/index.jsx | AI 对话界面 |
| `/image-recognition` | ImageRecognition.jsx | 图像识别界面 |
| `/file-manager` | FileManager/index.jsx | 文件管理界面 |

### 前端架构

前端采用模块化架构，核心模块包括：

- **AIChat 页面**：使用 Ant Design X 的 Bubble、Sender、Conversations 组件实现聊天功能
  - 支持流式响应 (SSE) 和普通响应
  - 支持“思考模式”开关，请求级切换普通模型与思考模型
  - 工具目录通过 `/api/v1/tools` 动态拉取，前端不再维护硬编码工具列表
  - SSE 直接消费后端返回的 `schema.Message` chunk，并按 `response_meta.finish_reason` 聚合多条产出消息
  - 当前正在接收的 SSE 回复保留逐字效果；历史消息和重新加载出的消息直接完整展示
  - 当当前轮次未启用工具且模型返回 `reasoning_content` 时，使用 Ant Design X `Think` 组件展示思考过程；开启思考模式的实时回复会先出现 `Think` loading 占位，并在最终回答正文开始前保持 loading
  - 当当前轮次启用工具时，使用 Ant Design X `ThoughtChain` 展示“工具规划 -> 工具调用 -> 工具结果 -> 最终回答”的链路，历史消息和实时流式共用同一套链路重建逻辑
  - 支持 TTS 语音合成
  - 支持文件上传 (.md/.txt)
  - 支持会话管理（创建、切换、删除、重命名）
  - 支持选择已索引的文件进行 RAG 对话

- **FileManager 页面**：文件管理界面
  - 文件列表展示（文件名、大小、类型、索引状态）
  - 文件上传功能
  - 手动触发文件索引
  - 删除文件索引
  - 删除文件（同时删除索引）
  - 索引状态管理（pending/indexing/indexed/failed）

---

## 常见开发任务

### 添加新的 AI 模型

1. 在 `common/aihelper/model.go` 中实现 `AIModel` 接口
2. 在 `factory.go` 中添加模型创建逻辑
3. 分配新的模型类型 ID

### 添加新的 API 接口

1. 在 `model/` 添加数据模型
2. 在 `dao/` 添加数据访问方法
3. 在 `service/` 添加业务逻辑
4. 在 `controller/` 添加 HTTP Handler
5. 在 `router/` 注册路由

### 添加 MCP 工具

1. 在 `common/mcp/server/server.go` 中使用 `mcpServer.AddTool()` 注册新工具
2. 在 `model.go` 的 `buildFirstPrompt()` 中添加工具描述
