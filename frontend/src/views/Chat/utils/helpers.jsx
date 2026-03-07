import { Avatar, Typography } from 'antd'
import { CopyOutlined, RedoOutlined, SoundOutlined, UserOutlined, RobotOutlined } from '@ant-design/icons'
import { Bubble, Actions } from '@ant-design/x'
import XMarkdown from '@ant-design/x-markdown'
import { COLORS, MESSAGE_MAX_WIDTH } from '../config/constants'

const { Text } = Typography

// ID 生成器
let idCounter = 0
export const generateMessageId = () => {
  idCounter += 1
  return `msg_${Date.now()}_${idCounter}`
}

// 创建消息操作项
export const createMessageActions = (isUser) => {
  const baseItems = [{ key: 'copy', icon: <CopyOutlined />, label: '复制' }]
  if (isUser) {
    return [...baseItems, { key: 'retry', icon: <RedoOutlined />, label: '重发' }]
  }
  return [...baseItems, { key: 'tts', icon: <SoundOutlined />, label: '朗读' }]
}

// Markdown 渲染
export const renderMarkdown = (content) => <XMarkdown content={content} />

// Role 配置
export const createRoleConfig = (onActionClick) => ({
  ai: {
    placement: 'start',
    avatar: <Avatar icon={<RobotOutlined />} style={{ backgroundColor: COLORS.primary }} />,
    header: <Text type="secondary" style={{ fontSize: 12 }}>AI 助手</Text>,
    contentRender: renderMarkdown,
    footer: (item) => item.loading ? null : (
      <Actions items={createMessageActions(false)} onClick={(info) => onActionClick(item, info)} />
    ),
    style: { maxWidth: MESSAGE_MAX_WIDTH }
  },
  user: {
    placement: 'end',
    typing: false,
    avatar: <Avatar icon={<UserOutlined />} style={{ backgroundColor: COLORS.success }} />,
    header: <Text type="secondary" style={{ fontSize: 12 }}>我</Text>,
    contentRender: renderMarkdown,
    footer: (item) => item.loading ? null : (
      <Actions items={createMessageActions(true)} onClick={(info) => onActionClick(item, info)} />
    ),
    style: { maxWidth: MESSAGE_MAX_WIDTH }
  },
  system: { placement: 'center', variant: 'borderless', avatar: null, style: { textAlign: 'center' } }
})

// 解析 SSE 数据行
export const parseSSELine = (line) => {
  const trimmedLine = line.trim()
  if (!trimmedLine || !trimmedLine.startsWith('data:')) {
    return null
  }

  const data = trimmedLine.slice(5).trim()

  if (data === '[DONE]') {
    return { type: 'done', data }
  }

  if (data.startsWith('{')) {
    try {
      const parsed = JSON.parse(data)
      return { type: 'json', data: parsed }
    } catch {
      return { type: 'text', data }
    }
  }

  return { type: 'text', data }
}
