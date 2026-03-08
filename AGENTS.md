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
│   ├── message.go          # 消息模型（含 index、role、tool_calls 字段）
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
│   ├── agent.go            # Agent 相关路由（统一接口）
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
- 根据请求中的 `tools` 参数动态创建 Agent
- 支持 `knowledge_search`、`sequential_thinking`、MCP 工具

**流式读取约定**:
- Agent 流式输出底层仍基于 `agent.Stream(ctx, messages)` 返回的 `*schema.StreamReader[*schema.Message]`
- `common/agent` 通过统一的流聚合器消费 `Recv()`，提取 `msg.Content`、`msg.ReasoningContent` 和 `msg.ToolCalls`
- 服务层负责把聚合结果转换为结构化事件流，并处理会话与持久化
- Controller 层由 `ChatHandler` 直接消费 service 返回的事件流，并统一调用 Gin `c.Stream(...)` + `c.SSEvent("message", payload)` 输出 SSE；channel 关闭时追加 `data: [DONE]`，客户端断连通过 `request.Context()` 继续向 service/Agent 链路传递取消信号
- 思维链与正文由不同 chunk 输出；服务端在收到第一个正文 chunk 前发送 `reasoning_end`
- 思考模式下，SSE 事件顺序必须保持为 `reasoning_delta* -> reasoning_end -> content_delta* -> message_end`

### 2. 消息存储流程

```
用户消息 → Agent 执行 → SSE 流式响应 → 保存消息到 Redis 缓存 → RabbitMQ 异步写入 PostgreSQL
```

消息通过 RabbitMQ 异步存储，避免阻塞主流程。
- Redis 缓存与 RabbitMQ 载荷都会保留 `index` 和 `tool_calls`，保证重新生成与消息回放一致

**消息模型**:
```go
type Message struct {
    ID        uint            // 主键
    SessionID string          // 会话 ID
    Index     int             // 线性索引（用于重新生成）
    Role      string          // user/assistant
    Content   string          // 消息内容
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
// 获取工具列表
tools := registry.GetToolsByNames(ctx, []string{"knowledge_search", "sequential_thinking"}, fileIDs)

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

**Agent 对话 (统一接口)**:
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/agent` | 发送消息/重新生成（支持流式/非流式） |
| GET | `/api/v1/agent/:session_id/messages` | 获取消息列表 |

**工具接口**:
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/tools` | 获取可用工具列表 |

**会话管理**:
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/sessions` | 获取会话列表 |
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

#### POST /api/v1/agent

**请求体**:
```json
{
  "session_id": "sess_001",           // [可选] 缺省时自动创建新会话
  "message": "帮我查一下天气",         // [可选] 用户新消息（重新生成时可为空）
  "regenerate_from": 3,               // [可选] 重新生成：从此索引截断
  "tools": ["knowledge_search", "sequential_thinking"], // [可选] 工具配置
  "thinking_mode": true,              // [可选] 是否启用思考模型
  "stream": true                      // [可选] 是否流式响应，默认 true
}
```

**响应体 (非流式)**:
```json
{
  "code": 1000,
  "msg": "success",
  "data": {
    "session_id": "sess_001",
    "message_index": 4,
    "role": "assistant",
    "reasoning_content": "先确认需求边界...",
    "content": "北京今天 25 度...",
    "tool_calls": [...]
  }
}
```

**SSE 流式事件格式**:
```
data: {"type": "meta", "session_id": "sess_001", "message_index": 4}
data: {"type": "reasoning_delta", "content": "先分析需求"}
data: {"type": "reasoning_end", "status": "completed"}
data: {"type": "tool_call", "tool_id": "knowledge_search", "function": "search", "arguments": {...}}
data: {"type": "content_delta", "content": "北"}
data: {"type": "content_delta", "content": "京"}
data: {"type": "message_end", "status": "completed"}
```

说明：
- 后端由 `common/agent` 统一消费 Eino `StreamReader` 的 `*schema.Message`
- HTTP SSE 由 Controller 直接消费 service 事件流并统一使用 Gin `c.Stream` 逐条输出；channel 关闭时追加 `data: [DONE]`，其余 `data` 中的 JSON 结构保持不变；客户端断连时取消信号会沿 `request.Context()` 继续传递给上游流式生成
- `reasoning_delta` 与 `content_delta` 不允许在前端可见层面并行输出；正文必须在 `reasoning_end` 之后开始
- 前端展示层在本地打字动画结束前持续使用逐字内容，不允许在 `reasoning_end` 或 `message_end` 到达时直接切回完整文本

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
  - 当模型返回 `reasoning_content` / `reasoning_delta` 时，使用 Ant Design X `Think` 组件展示思考过程
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
