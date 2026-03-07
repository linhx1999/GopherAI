/**
 * AIChat 常量配置
 * 集中管理所有魔法值和配置项
 */

// ==================== 分页配置 ====================
export const MESSAGE_PAGE_SIZE = 10

// ==================== 打字效果配置 ====================
export const TYPING_CONFIG = {
  step: 2,
  interval: 20,
}

// ==================== TTS 轮询配置 ====================
export const TTS_CONFIG = {
  initialWaitTime: 5000,
  maxPollAttempts: 30,
  pollInterval: 2000,
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

// ==================== 工具选项 ====================
export const TOOL_OPTIONS = [
  { label: '知识库检索', value: 'knowledge_search', icon: 'SearchOutlined' },
  { label: '逐步思考', value: 'sequential_thinking', icon: 'BulbOutlined' },
  { label: '天气查询', value: 'get_weather', icon: 'CloudOutlined' },
]

// ==================== 会话菜单配置 ====================
export const SESSION_MENU_ITEMS = [
  { label: '重命名', key: 'rename', icon: 'EditOutlined' },
  { label: '分享', key: 'share', icon: 'ShareAltOutlined' },
  { type: 'divider' },
  { label: '删除会话', key: 'deleteChat', icon: 'DeleteOutlined', danger: true }
]

// ==================== API 配置 ====================
export const API_ENDPOINTS = {
  // Agent 接口 - 统一的 RESTful 接口
  AGENT: 'agent',                                      // POST: 发送消息/重新生成
  AGENT_MESSAGES: (sessionId) => `agent/${sessionId}/messages`, // GET: 获取消息列表

  // 工具接口
  TOOLS: 'tools',                                      // GET: 获取工具列表

  // 会话接口
  SESSIONS: 'sessions',                                // GET: 获取会话列表
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

  // TTS 接口
  TTS: 'tts',
  TTS_QUERY: 'tts/query',
}

// ==================== SSE 事件类型 ====================
export const SSE_EVENT_TYPES = {
  META: 'meta',
  TOOL_CALL: 'tool_call',
  CONTENT_DELTA: 'content_delta',
  MESSAGE_END: 'message_end',
  ERROR: 'error',
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
  SYSTEM: 'system',
}

// ==================== 特殊会话 ID ====================
export const SPECIAL_SESSIONS = {
  TEMP: 'temp',
}
