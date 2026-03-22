import { Avatar, Typography } from 'antd'
import { CopyOutlined, SoundOutlined, UserOutlined } from '@ant-design/icons'
import XMarkdown from '@ant-design/x-markdown'
import {
  ASSISTANT_DISPLAY_MODES,
  COLORS,
  MESSAGE_MAX_WIDTH
} from '../config/constants'

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

const DEFAULT_TOOL_DISPLAY_NAMES = {
  knowledge_search: '知识库检索',
  sequential_thinking: '逐步思考'
}

const hasVisibleMessageText = (message = {}) => Boolean(
  String(message?.content || '').trim() || String(message?.reasoning_content || '').trim()
)

const getMessageDisplayContent = (message = {}) => {
  const reasoning = String(message?.reasoning_content || '').trim()
  const content = String(message?.content || '').trim()

  if (reasoning && content) {
    return `${reasoning}\n\n${content}`
  }

  return reasoning || content
}

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

export const mergeSchemaMessageChunk = (baseMessage = {}, chunk = {}) => {
  const safeBaseMessage = baseMessage || {}
  const safeChunk = chunk || {}

  return {
    ...safeBaseMessage,
    ...safeChunk,
    content: `${safeBaseMessage.content || ''}${safeChunk.content || ''}`,
    reasoning_content: `${safeBaseMessage.reasoning_content || ''}${safeChunk.reasoning_content || ''}`,
    multi_content: safeChunk.multi_content?.length ? [...(safeBaseMessage.multi_content || []), ...safeChunk.multi_content] : (safeBaseMessage.multi_content || []),
    user_input_multi_content: safeChunk.user_input_multi_content?.length ? [...(safeBaseMessage.user_input_multi_content || []), ...safeChunk.user_input_multi_content] : (safeBaseMessage.user_input_multi_content || []),
    assistant_output_multi_content: safeChunk.assistant_output_multi_content?.length ? [...(safeBaseMessage.assistant_output_multi_content || []), ...safeChunk.assistant_output_multi_content] : (safeBaseMessage.assistant_output_multi_content || []),
    tool_calls: mergeToolCalls(safeBaseMessage.tool_calls || [], safeChunk.tool_calls || []),
    response_meta: safeChunk.response_meta || safeBaseMessage.response_meta,
  }
}

export const isSchemaMessagePayload = (data) => Boolean(data?.role)

export const isMessageFinished = (message) => Boolean(message?.response_meta?.finish_reason)

export const normalizeEnabledToolAPINames = (toolAPINames) => (
  Array.isArray(toolAPINames)
    ? [...new Set(toolAPINames.map((toolName) => String(toolName || '').trim()).filter(Boolean))].sort()
    : []
)

export const formatToolArguments = (argumentsText) => {
  if (!argumentsText) return ''
  try {
    return JSON.stringify(JSON.parse(argumentsText), null, 2)
  } catch {
    return argumentsText
  }
}

export const createToolDisplayNameMap = (toolCatalog = []) => {
  const labels = { ...DEFAULT_TOOL_DISPLAY_NAMES }

  toolCatalog.forEach((tool) => {
    const apiName = String(tool?.apiName || tool?.name || '').trim()
    if (!apiName) {
      return
    }
    labels[apiName] = tool.displayName || tool.display_name || labels[apiName] || apiName
  })

  return labels
}

export const getToolDisplayName = (toolName, toolDisplayNames = DEFAULT_TOOL_DISPLAY_NAMES) => (
  toolDisplayNames[toolName] || DEFAULT_TOOL_DISPLAY_NAMES[toolName] || toolName || '工具'
)

const isToolTraceMessage = (message = {}) => {
  if (message.role === 'tool') return true
  if (message.role !== 'assistant') return false
  const hasVisibleAnswer = hasVisibleMessageText(message)
  return !hasVisibleAnswer && Array.isArray(message.tool_calls) && message.tool_calls.length > 0
}

const getThoughtChainDescription = (descriptionMessage = null) => {
  if (!descriptionMessage) {
    return ''
  }

  return getMessageDisplayContent(descriptionMessage)
}

export const hasToolActivity = (toolTraceRecords = []) => toolTraceRecords.some((record) => {
  const message = record.message || {}
  return message.role === 'tool' || (Array.isArray(message.tool_calls) && message.tool_calls.length > 0)
})

export const shouldRenderToolTrace = (record, toolTraceRecords = []) => {
  if (!record || record.message?.role !== 'assistant') {
    return false
  }

  if (record.assistantRenderMode === ASSISTANT_DISPLAY_MODES.TOOL_CHAIN) {
    return true
  }

  return hasToolActivity(toolTraceRecords)
}

const buildToolCallItems = ({ record, toolTraceRecords = [] }) => toolTraceRecords.flatMap((traceRecord, traceIndex) => {
  const message = traceRecord.message || {}
  const toolCalls = Array.isArray(message.tool_calls) ? message.tool_calls : []

  return toolCalls.map((call, index) => {
    const toolName = call.function?.name || call.id || `tool-${index + 1}`
    const formattedArguments = formatToolArguments(call.function?.arguments)
    const traceDescription = getThoughtChainDescription(
      traceRecord.traceDescription || (traceIndex === 0 ? record?.planningMessage : null)
    )

    return {
      key: `${traceRecord.key}-call-${call.id || call.index || index}`,
      title: toolName,
      description: traceDescription || (formattedArguments ? '参数已准备' : '等待工具执行'),
      content: formattedArguments ? (
        <pre className="thought-chain-code-block">{formattedArguments}</pre>
      ) : null,
      collapsible: Boolean(formattedArguments),
      status: traceRecord.pending ? 'loading' : 'success',
      blink: Boolean(traceRecord.pending)
    }
  })
})

const buildToolResultItems = (toolTraceRecords = []) => toolTraceRecords.flatMap((record, index) => {
  const message = record.message || {}
  if (message.role !== 'tool') {
    return []
  }

  const toolName = message.tool_name || message.name || `tool-result-${index + 1}`
  const content = message.content || ''

  return [{
    key: `${record.key}-result`,
    title: `${toolName} 执行结果`,
    description: content ? '工具返回了可展示内容' : '工具已完成执行',
    content: content ? (
      <div className="thought-chain-markdown">
        {renderMarkdown(content)}
      </div>
    ) : null,
    collapsible: Boolean(content),
    status: record.pending ? 'loading' : 'success',
    blink: Boolean(record.pending)
  }]
})

export const buildToolTraceItems = ({ record = null, toolTraceRecords = [] }) => {
  const toolCallItems = buildToolCallItems({ record, toolTraceRecords })
  const toolResultItems = buildToolResultItems(toolTraceRecords)

  return [
    ...toolCallItems,
    ...toolResultItems
  ].filter(Boolean)
}

export const buildDisplayMessages = (records) => {
  const display = []
  let toolTraceBuffer = []

  records.forEach((record) => {
    const message = record.message || {}

    if (message.role === 'user' || message.role === 'system') {
      if (toolTraceBuffer.length > 0) {
        display.push({ type: 'tool_trace', key: `tool-trace-${toolTraceBuffer[0].key}`, toolTraceRecords: toolTraceBuffer })
        toolTraceBuffer = []
      }
      display.push({ type: message.role, key: record.key, record })
      return
    }

    if (
      message.role === 'assistant' &&
      (
        record.assistantRenderMode === ASSISTANT_DISPLAY_MODES.TOOL_CHAIN ||
        record.planningMessage ||
        (Array.isArray(record.toolTraceRecords) && record.toolTraceRecords.length > 0)
      )
    ) {
      if (toolTraceBuffer.length > 0) {
        display.push({ type: 'tool_trace', key: `tool-trace-${toolTraceBuffer[0].key}`, toolTraceRecords: toolTraceBuffer })
        toolTraceBuffer = []
      }

      display.push({
        type: 'assistant',
        key: record.key,
        record,
        toolTraceRecords: record.toolTraceRecords || []
      })
      return
    }

    if (isToolTraceMessage(message)) {
      toolTraceBuffer.push(record)
      return
    }

    if (message.role === 'assistant') {
      display.push({
        type: 'assistant',
        key: record.key,
        record,
        toolTraceRecords: toolTraceBuffer
      })
      toolTraceBuffer = []
      return
    }

    toolTraceBuffer.push(record)
  })

  if (toolTraceBuffer.length > 0) {
    display.push({ type: 'tool_trace', key: `tool-trace-${toolTraceBuffer[0].key}`, toolTraceRecords: toolTraceBuffer })
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
