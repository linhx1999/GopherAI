/**
 * AIChat 常量配置
 * 集中管理所有魔法值和配置项
 */

// ==================== 分页配置 ====================
export const MESSAGE_PAGE_SIZE = 10
export const MESSAGE_LIST_BOTTOM_THRESHOLD = 120

// ==================== 打字效果配置 ====================
export const TYPING_CONFIG = {
  step: 2,
  interval: 20,
}

// ==================== 文件上传配置 ====================
export const SUPPORTED_FILE_TYPES = ['.md', '.txt']
export const SUPPORTED_MIME_TYPES = ['text/markdown', 'text/plain']

// ==================== 样式常量 ====================
export const COLORS = {
  primary: '#1677ff',
  success: '#52c41a',
  error: '#ff4d4f',
  warning: '#faad14',
}

export const MESSAGE_MAX_WIDTH = '80%'

// ==================== 会话菜单配置 ====================
export const SESSION_MENU_ITEMS = [
  { label: '重命名', key: 'rename', icon: 'EditOutlined' },
  { label: '分享', key: 'share', icon: 'ShareAltOutlined' },
  { type: 'divider' },
  { label: '删除会话', key: 'deleteChat', icon: 'DeleteOutlined', danger: true }
]

// ==================== API 配置 ====================
export const API_ENDPOINTS = {
  // Agent 接口
  AGENT_GENERATE: 'agent/generate',                    // POST: 非流式生成
  AGENT_STREAM: 'agent/stream',                        // POST: 流式生成
  AGENT_MESSAGES: (sessionId) => `agent/${sessionId}/messages`, // GET: 获取消息列表
  DEEP_AGENT_GENERATE: 'deep-agent/generate',
  DEEP_AGENT_STREAM: 'deep-agent/stream',
  DEEP_AGENT_RUNTIME: 'deep-agent/runtime',
  DEEP_AGENT_RUNTIME_RESTART: 'deep-agent/runtime/restart',
  DEEP_AGENT_RUNTIME_REBUILD: 'deep-agent/runtime/rebuild',

  // 工具接口
  TOOLS: 'tools',                                      // GET: 获取工具列表

  // MCP 接口
  MCP_SERVERS: 'mcp/servers',
  MCP_SERVER_DETAIL: (serverId) => `mcp/servers/${serverId}`,
  MCP_SERVER_TEST: (serverId) => `mcp/servers/${serverId}/test`,

  // 会话接口
  SESSIONS: 'sessions',                                // GET: 获取会话列表 / POST: 创建会话
  SESSION_TITLE: (sessionId) => `sessions/${sessionId}/title`, // PUT: 更新标题
  SESSION_DELETE: (sessionId) => `sessions/${sessionId}`,      // DELETE: 删除会话

  // 文件接口
  FILE_UPLOAD: 'file/upload',
  FILE_LIST: 'file/list',
  FILE_DELETE: (fileId) => `file/${fileId}`,
  FILE_URL: (fileId) => `file/url/${fileId}`,
  FILE_DOWNLOAD: (fileId) => `file/download/${fileId}`,
  FILE_INDEX: (fileId) => `file/index/${fileId}`,
  FILE_INDEX_DELETE: (fileId) => `file/index/${fileId}`,
}

// ==================== SSE 事件类型 ====================
export const SSE_EVENT_TYPES = {
  RESPONSE_CREATED: 'response.created',
  RESPONSE_MESSAGE_DELTA: 'response.message.delta',
  RESPONSE_MESSAGE_COMPLETED: 'response.message.completed',
  RESPONSE_ERROR: 'response.error',
  RESPONSE_DONE: 'response.done',
}

// ==================== 状态码 ====================
export const STATUS_CODES = {
  SUCCESS: 1000,
}

// ==================== 消息角色 ====================
export const MESSAGE_ROLES = {
  USER: 'user',
  AI: 'ai',
  ASSISTANT: 'assistant',
  TOOL: 'tool',
  SYSTEM: 'system',
}

export const ASSISTANT_DISPLAY_MODES = {
  DEFAULT: 'default',
  THOUGHT_CHAIN: 'thought_chain',
  TOOL_CHAIN: 'thought_chain',
}

export const TOOL_TRACE_KINDS = {
  REASONING: 'reasoning',
  CALL: 'tool_call',
  RESULT: 'tool_result',
}

export const TOOL_TRACE_STATUS = {
  LOADING: 'loading',
  SUCCESS: 'success',
  ERROR: 'error',
}

// ==================== 特殊会话 ID ====================
export const SPECIAL_SESSIONS = {
  TEMP: 'temp',
}

export const AGENT_MODES = {
  CHAT: 'chat',
  DEEP: 'deep',
}
