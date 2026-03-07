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

// ==================== 模型选项 ====================
export const MODEL_OPTIONS = [
  { label: 'Chat', value: '1' },
  { label: 'RAG', value: '2' },
  { label: 'MCP', value: '3' },
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
  CHAT_STREAM: '/AI/chat/send-stream',
  CHAT_STREAM_NEW_SESSION: '/AI/chat/send-stream-new-session',
  CHAT_SEND: '/AI/chat/send',
  CHAT_SEND_NEW_SESSION: '/AI/chat/send-new-session',
  CHAT_HISTORY: '/AI/chat/history',
  SESSIONS: '/AI/chat/sessions',
  SESSION_TITLE: (sessionId) => `/AI/chat/sessions/${sessionId}/title`,
  SESSION_DELETE: (sessionId) => `/AI/chat/sessions/${sessionId}`,
  SESSION_SUMMARIZE: (sessionId) => `/AI/chat/sessions/${sessionId}/summarize`,
  TTS: '/AI/chat/tts',
  TTS_QUERY: '/AI/chat/tts/query',
  FILE_UPLOAD: '/file/upload',
  FILE_LIST: '/file/list',
  FILE_DELETE: (fileId) => `/file/${fileId}`,
  FILE_URL: (fileId) => `/file/url/${fileId}`,
  FILE_DOWNLOAD: (fileId) => `/file/download/${fileId}`,
  FILE_INDEX: (fileId) => `/file/index/${fileId}`,
  FILE_INDEX_DELETE: (fileId) => `/file/index/${fileId}`,
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
