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
  const hasVisibleAnswer = Boolean(message.content || message.reasoning_content)
  return !hasVisibleAnswer && Array.isArray(message.tool_calls) && message.tool_calls.length > 0
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

const buildPlanningItem = ({ record, hasExecution, toolDisplayNames }) => {
  if (!record?.assistantRenderMode && !hasExecution) {
    return null
  }

  const enabledToolAPINames = normalizeEnabledToolAPINames(record?.enabledToolAPINames)
  const isPending = Boolean(record?.pending) && !hasExecution
  const description = enabledToolAPINames.length > 0
    ? `已启用：${enabledToolAPINames.map((toolName) => getToolDisplayName(toolName, toolDisplayNames)).join('、')}`
    : (hasExecution ? '已进入工具执行链路' : '工具规划中')

  return {
    key: `${record?.key || 'tool-trace'}-plan`,
    title: '工具规划',
    description,
    status: isPending ? 'loading' : 'success',
    blink: isPending
  }
}

const buildToolCallItems = (toolTraceRecords = [], toolDisplayNames) => toolTraceRecords.flatMap((record) => {
  const message = record.message || {}
  const toolCalls = Array.isArray(message.tool_calls) ? message.tool_calls : []

  return toolCalls.map((call, index) => {
    const toolName = call.function?.name || call.id || `tool-${index + 1}`
    const formattedArguments = formatToolArguments(call.function?.arguments)

    return {
      key: `${record.key}-call-${call.id || call.index || index}`,
      title: `调用 ${getToolDisplayName(toolName, toolDisplayNames)}`,
      description: formattedArguments ? '参数已准备' : '等待工具执行',
      content: formattedArguments ? (
        <pre className="thought-chain-code-block">{formattedArguments}</pre>
      ) : null,
      collapsible: Boolean(formattedArguments),
      status: record.pending ? 'loading' : 'success',
      blink: Boolean(record.pending)
    }
  })
})

const buildToolResultItems = (toolTraceRecords = [], toolDisplayNames) => toolTraceRecords.flatMap((record, index) => {
  const message = record.message || {}
  if (message.role !== 'tool') {
    return []
  }

  const toolName = message.tool_name || message.name || `tool-result-${index + 1}`
  const content = message.content || ''

  return [{
    key: `${record.key}-result`,
    title: `${getToolDisplayName(toolName, toolDisplayNames)} 执行结果`,
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

const buildAnswerItem = ({ record, hasExecution }) => {
  if (!record) {
    return null
  }

  const message = record.message || {}
  const hasAnswer = Boolean((message.content || '').trim())
  const isPending = Boolean(record.pending) && !hasAnswer

  return {
    key: `${record.key}-answer`,
    title: hasExecution ? '整理最终回答' : '直接生成回答',
    description: hasAnswer ? '已生成最终回答' : (isPending ? '正在生成最终回答' : '当前轮次未返回正文'),
    status: isPending ? 'loading' : 'success',
    blink: isPending
  }
}

export const buildToolTraceItems = ({ record = null, toolTraceRecords = [], toolDisplayNames = DEFAULT_TOOL_DISPLAY_NAMES }) => {
  const hasExecution = hasToolActivity(toolTraceRecords)
  const planningItem = buildPlanningItem({ record, hasExecution, toolDisplayNames })
  const toolCallItems = buildToolCallItems(toolTraceRecords, toolDisplayNames)
  const toolResultItems = buildToolResultItems(toolTraceRecords, toolDisplayNames)
  const answerItem = buildAnswerItem({ record, hasExecution })

  return [
    planningItem,
    ...toolCallItems,
    ...toolResultItems,
    answerItem
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
