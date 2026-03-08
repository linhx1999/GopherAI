import { Avatar, Typography } from 'antd'
import { CopyOutlined, SoundOutlined, UserOutlined } from '@ant-design/icons'
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
    return baseItems
  }
  return [...baseItems, { key: 'tts', icon: <SoundOutlined />, label: '朗读' }]
}

// Markdown 渲染
export const renderMarkdown = (content) => <XMarkdown content={content} />

// Role 配置
export const createRoleConfig = () => ({
  user: {
    placement: 'end',
    typing: false,
    avatar: <Avatar icon={<UserOutlined />} style={{ backgroundColor: COLORS.success }} />,
    header: <Text type="secondary" style={{ fontSize: 12 }}>我</Text>,
    contentRender: renderMarkdown,
    style: { maxWidth: MESSAGE_MAX_WIDTH }
  },
  system: { placement: 'center', variant: 'borderless', avatar: null, style: { textAlign: 'center' } }
})

const mergeToolCalls = (baseCalls = [], nextCalls = []) => {
  const merged = baseCalls.map((call) => ({
    ...call,
    function: call.function ? { ...call.function } : undefined
  }))

  nextCalls.forEach((call, seq) => {
    const index = merged.findIndex((item, itemSeq) => {
      if (call.index !== undefined && item.index !== undefined) {
        return item.index === call.index
      }
      if (call.id && item.id) {
        return item.id === call.id
      }
      return itemSeq === seq
    })

    if (index === -1) {
      merged.push({
        ...call,
        function: call.function ? { ...call.function } : undefined
      })
      return
    }

    const previous = merged[index]
    merged[index] = {
      ...previous,
      ...call,
      function: {
        ...previous.function,
        ...call.function,
        arguments: `${previous.function?.arguments || ''}${call.function?.arguments || ''}`
      }
    }
  })

  return merged
}

export const mergeSchemaMessageChunk = (baseMessage = {}, chunk = {}) => ({
  ...baseMessage,
  ...chunk,
  content: `${baseMessage.content || ''}${chunk.content || ''}`,
  reasoning_content: `${baseMessage.reasoning_content || ''}${chunk.reasoning_content || ''}`,
  multi_content: chunk.multi_content?.length ? [...(baseMessage.multi_content || []), ...chunk.multi_content] : (baseMessage.multi_content || []),
  user_input_multi_content: chunk.user_input_multi_content?.length ? [...(baseMessage.user_input_multi_content || []), ...chunk.user_input_multi_content] : (baseMessage.user_input_multi_content || []),
  assistant_output_multi_content: chunk.assistant_output_multi_content?.length ? [...(baseMessage.assistant_output_multi_content || []), ...chunk.assistant_output_multi_content] : (baseMessage.assistant_output_multi_content || []),
  tool_calls: mergeToolCalls(baseMessage.tool_calls || [], chunk.tool_calls || []),
  response_meta: chunk.response_meta || baseMessage.response_meta,
})

export const isSchemaMessagePayload = (data) => Boolean(data?.role)

export const isMessageFinished = (message) => Boolean(message?.response_meta?.finish_reason)

export const formatToolArguments = (argumentsText) => {
  if (!argumentsText) return ''
  try {
    return JSON.stringify(JSON.parse(argumentsText), null, 2)
  } catch {
    return argumentsText
  }
}

const isProcessMessage = (message = {}) => {
  if (message.role === 'tool') return true
  if (message.role !== 'assistant') return false
  const hasVisibleAnswer = Boolean(message.content || message.reasoning_content)
  return !hasVisibleAnswer && Array.isArray(message.tool_calls) && message.tool_calls.length > 0
}

export const buildDisplayMessages = (records) => {
  const display = []
  let processBuffer = []

  records.forEach((record) => {
    const message = record.message || {}

    if (message.role === 'user' || message.role === 'system') {
      if (processBuffer.length > 0) {
        display.push({ type: 'process', key: `process-${processBuffer[0].key}`, processes: processBuffer })
        processBuffer = []
      }
      display.push({ type: message.role, key: record.key, record })
      return
    }

    if (isProcessMessage(message)) {
      processBuffer.push(record)
      return
    }

    if (message.role === 'assistant') {
      display.push({
        type: 'assistant',
        key: record.key,
        record,
        processes: processBuffer
      })
      processBuffer = []
      return
    }

    processBuffer.push(record)
  })

  if (processBuffer.length > 0) {
    display.push({ type: 'process', key: `process-${processBuffer[0].key}`, processes: processBuffer })
  }

  return display
}

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
