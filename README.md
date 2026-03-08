# GopherAI

一个基于 Go 语言开发的 AI 助手平台，提供多模型 AI 对话、图像识别、语音合成、RAG 知识库检索等功能。

## 功能特性

- **多模型 AI 对话** - 支持 OpenAI 兼容模型、Ollama 本地模型、阿里云 RAG 模型
- **流式响应** - 支持 SSE 流式输出，实时展示 AI 回复；历史消息回放时直接完整展示，不再逐字重放
- **思考模式** - 支持在聊天页切换普通模型与思考模型，并在模型返回推理内容时展示思考过程；实时思考回复会在最终回答输出前保持 loading，占位期间和推理输出阶段都可见
- **RAG 知识库** - 上传文档构建知识库，基于向量检索增强生成
  - 支持文档切分（Markdown 标题切分、文本固定长度切分）
  - 支持指定文件进行 RAG 对话
  - 使用 PostgreSQL pgvector 进行向量存储和检索
  - IVFFlat 索引 + 余弦相似度
  - 索引状态管理
- **文件管理** - 独立的文件管理界面
  - 文件上传和下载
  - 手动触发文件索引
  - 索引状态监控
  - 文件和索引删除
- **MCP 工具调用** - 支持 Model Context Protocol，扩展 AI 能力
- **图像识别** - 上传图片进行 AI 分析识别
- **语音合成 (TTS)** - 将文本转换为语音输出
- **用户认证** - JWT 身份验证，安全的会话管理

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端框架 | Go 1.25 + Gin |
| 前端框架 | React 19 + Ant Design 6 + Ant Design X |
| 构建工具 | Vite 7 |
| ORM | GORM |
| 数据库 | PostgreSQL + pgvector |
| 缓存 | Redis |
| 消息队列 | RabbitMQ |
| 对象存储 | MinIO |
| AI 框架 | CloudWeGo Eino |
| 协议 | MCP (Model Context Protocol) |

## 快速开始

### 环境要求

- Go 1.25+
- Node.js 16+
- PostgreSQL 16+ (with pgvector extension)
- Redis 7+
- RabbitMQ 3.x

### 安装步骤

**1. 克隆项目**

```bash
git clone https://github.com/your-username/GopherAI.git
cd GopherAI
```

**2. 配置环境变量**

```bash
cp .env.example .env
# 编辑 .env 文件，填写实际配置值
```

**3. 启动依赖服务**

```bash
# 使用 Docker Compose (推荐)
docker-compose up -d

# 或手动启动 PostgreSQL、Redis、RabbitMQ
```

**4. 启动后端服务**

```bash
go mod download
go run main.go
```

**5. 启动前端服务**

```bash
cd frontend
pnpm install
pnpm dev
```

**6. 访问应用**

打开浏览器访问 `http://localhost:8080`

## 配置说明

配置通过 `.env` 环境变量文件管理，主要配置项：

### 应用配置

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `APP_NAME` | 应用名称 | GopherAI |
| `APP_HOST` | 监听地址 | 0.0.0.0 |
| `APP_PORT` | 服务端口 | 9090 |

### 数据库配置

| 变量 | 说明 |
|------|------|
| `POSTGRES_HOST` | PostgreSQL 主机地址 |
| `POSTGRES_PORT` | PostgreSQL 端口 |
| `POSTGRES_USER` | 用户名 |
| `POSTGRES_PASSWORD` | 密码 |
| `POSTGRES_DB` | 数据库名 |
| `POSTGRES_SSL_MODE` | SSL 模式 |

### AI 模型配置

| 变量 | 说明 |
|------|------|
| `OPENAI_API_KEY` | API 密钥 |
| `OPENAI_MODEL_NAME` | 默认对话模型名称 |
| `OPENAI_REASONING_MODEL_NAME` | 思考模式使用的推理模型名称 |
| `OPENAI_BASE_URL` | API 基础 URL |

### RAG 配置

| 变量 | 说明 |
|------|------|
| `RAG_EMBEDDING_MODEL` | 嵌入模型 |
| `RAG_CHAT_MODEL` | 对话模型 |
| `RAG_DOC_DIR` | 文档目录 |
| `RAG_DIMENSION` | 向量维度 |

### MinIO 配置

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `MINIO_ENDPOINT` | MinIO 服务地址 | localhost:9000 |
| `MINIO_ACCESS_KEY` | 访问密钥 | minioadmin |
| `MINIO_SECRET_KEY` | 秘密密钥 | minioadmin |
| `MINIO_BUCKET` | 存储桶名称 | gopherai |
| `MINIO_USE_SSL` | 是否使用 SSL | false |

## API 文档

### 认证接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/user/register` | 用户注册 |
| POST | `/api/v1/user/login` | 用户登录 |

### AI 对话接口 (需认证)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/agent/generate` | 非流式生成消息 |
| POST | `/api/v1/agent/stream` | SSE 流式生成消息 |
| GET | `/api/v1/agent/:session_id/messages` | 获取当前用户的会话消息 |
| GET | `/api/v1/tools` | 获取可用工具列表 |

### 其他接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/image/recognize` | 图像识别 |
| POST | `/api/v1/file/upload` | 上传文件 |
| GET | `/api/v1/file/list` | 获取文件列表 |
| GET | `/api/v1/file/url/:id` | 获取文件访问 URL |
| GET | `/api/v1/file/download/:id` | 下载文件 |
| DELETE | `/api/v1/file/:id` | 删除文件 |
| POST | `/api/v1/file/index/:id` | 手动触发文件索引 |
| DELETE | `/api/v1/file/index/:id` | 删除文件索引 |

## 项目结构

```
GopherAI/
├── main.go              # 入口文件
├── config/              # 配置管理
├── model/               # 数据模型
│   ├── user.go          # 用户模型
│   ├── session.go       # 会话模型
│   ├── message.go       # 消息模型
│   ├── file.go          # 文件模型
│   └── document_chunk.go# 文档分块模型（RAG 向量存储）
├── dao/                 # 数据访问层
│   ├── user/
│   ├── session/
│   ├── message/
│   └── file/
├── service/             # 业务逻辑层
│   ├── user/
│   ├── session/
│   ├── image/
│   ├── file/
│   └── rag/             # RAG 索引服务
├── controller/          # 控制器层
├── router/              # 路由注册
├── middleware/          # 中间件
├── common/              # 公共组件
│   ├── aihelper/        # AI 助手核心模块
│   ├── rag/             # RAG 检索增强
│   ├── mcp/             # MCP 服务
│   ├── postgres/        # PostgreSQL + pgvector
│   ├── minio/           # MinIO 对象存储
│   └── ...
├── utils/               # 工具函数
└── frontend/            # React 前端项目
    └── src/
        ├── views/       # 页面组件
        │   ├── AIChat/  # AI 对话页面
        │   ├── FileManager/# 文件管理页面
        │   ├── Login.jsx
        │   ├── Register.jsx
        │   ├── Menu.jsx
        │   └── ImageRecognition.jsx
        ├── router/      # 前端路由
        └── utils/api.js # API 封装
```

## 思考模式

前端聊天输入区提供“思考”开关：

- 关闭时，请求使用 `OPENAI_MODEL_NAME`
- 打开时，请求使用 `OPENAI_REASONING_MODEL_NAME`
- 如果模型在响应中返回推理内容，前端会用 `Think` 组件实时展示思考过程
- 开启思考模式的实时回复会先显示 `Think` loading 占位，并在最终回答正文开始输出前保持 loading；历史消息则直接展示完整内容
- 如果模型未返回推理内容，则只展示最终回答

### Agent Generate 请求示例

```json
{
  "session_id": "sess_001",
  "message": "帮我分析一下这个方案",
  "tools": ["knowledge_search"],
  "thinking_mode": true
}
```

### Agent Generate 响应示例

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
      "content": "可以按接口职责拆分为 generate 和 stream。",
      "tool_calls": []
    }
  }
}
```

### Agent Stream 请求示例

```json
{
  "session_id": "sess_001",
  "message": "帮我分析一下这个方案",
  "tools": ["knowledge_search"],
  "thinking_mode": true
}
```

### SSE 事件

流式响应保留一个首包元数据事件，其余 `data` 直接发送完整的 `schema.Message` JSON：

```text
data: {"type":"meta","session_id":"sess_001","message_index":4}
data: {"role":"assistant","reasoning_content":"先确认问题约束..."}
data: {"role":"assistant","content":"答案","response_meta":{"finish_reason":"stop"}}
data: {"role":"tool","tool_name":"knowledge_search","content":"...","response_meta":{"finish_reason":"stop"}}
```

后端基于 Eino 的 `react.WithMessageFuture()` 捕获 Agent 过程中的每条产出消息流。`common/agent` 逐条消费 `MessageFuture.GetMessageStreams()` 返回的 `*schema.StreamReader[*schema.Message]`，原样向前端转发 `schema.Message` chunk，并在每条产出消息结束后用 `schema.ConcatMessages()` 聚合完整消息；如果底层未返回 `response_meta.finish_reason`，后端会补一个终止 chunk 以便前端稳定切分消息边界。`controller` 中的 `StreamHandler` 直接消费 service 事件流并通过 Gin `c.Stream(...)` + `c.SSEvent("message", payload)` 逐条输出 SSE；channel 关闭时追加 `data: [DONE]` 结束标记，请求断连会通过 `request.Context()` 向上游传播取消信号。前端收到 `schema.Message` 后会分别累积 `reasoning_content` 和 `content`，并把工具调用/工具结果消息归入折叠的执行过程卡片。
消息缓存和异步落库都会保留 `index`、`payload` 与 `tool_calls`，历史消息接口返回 `{index, message, created_at}`，其中 `message` 为完整 `schema.Message`，保证刷新回放和直播展示一致。

### 历史消息接口示例

`GET /api/v1/agent/:session_id/messages`

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

- 仅允许读取当前用户自己的会话历史；`session_id` 不存在或不属于当前用户时统一返回 `code=2011`
- 前端直接读取 `data.messages`，不再使用 `data[0].messages`

### 会话列表接口示例

`GET /api/v1/sessions`

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

- 只返回当前用户自己的会话
- 前端直接读取 `data.sessions`，并按 `createdAt` 排序

## 支持的 AI 模型

| 类型 | 说明 |
|------|------|
| OpenAI 兼容 | 支持 OpenAI、DeepSeek、通义千问等兼容 API |
| Ollama | 本地部署的开源模型 |
| RAG | 阿里云 RAG 模型，支持知识库检索 |
| MCP | 集成外部工具的模型 |

## 部署

### Docker 部署

```bash
# 构建镜像
docker build -t gopherai .

# 运行容器
docker run -d -p 9090:9090 --env-file .env gopherai
```

### 编译部署

```bash
# 编译
go build -o gopherai main.go

# 运行
./gopherai
```

## License

MIT License
