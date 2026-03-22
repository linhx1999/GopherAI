import { Avatar, Typography } from 'antd'
import { CopyOutlined, SoundOutlined, UserOutlined } from '@ant-design/icons'
import XMarkdown from '@ant-design/x-markdown'
import {
  ASSISTANT_DISPLAY_MODES,
  COLORS,
  MESSAGE_MAX_WIDTH,
  TOOL_TRACE_KINDS,
  TOOL_TRACE_STATUS
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
  sequentialthinking: '逐步思考'
}

export const hasVisibleMessageText = (message = {}) => Boolean(
  String(message?.content || '').trim() || String(message?.reasoning_content || '').trim()
)

export const getMessageDisplayContent = (message = {}) => {
  const reasoning = String(message?.reasoning_content || '').trim()
  const content = String(message?.content || '').trim()

  if (reasoning && content) {
    return `${reasoning}\n\n${content}`
  }

  return reasoning || content
}

export const getAssistantPlainContent = (message = {}) => String(message?.content || '').trim()

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

const formatToolCodeBlockMarkdown = (content) => {
  if (!content) {
    return ''
  }

  let formattedContent = content
  let language = ''

  try {
    formattedContent = JSON.stringify(JSON.parse(content), null, 2)
    language = 'json'
  } catch {
    formattedContent = content
  }

  return `\`\`\`${language}\n${formattedContent}\n\`\`\``
}

export const formatToolArgumentsMarkdown = (argumentsText) => {
  return formatToolCodeBlockMarkdown(argumentsText)
}

export const formatToolResultMarkdown = (resultText) => {
  return formatToolCodeBlockMarkdown(resultText)
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

export const hasToolCalls = (message = {}) => Array.isArray(message?.tool_calls) && message.tool_calls.length > 0

export const isToolCallAssistantMessage = (message = {}) => (
  message?.role === 'assistant' &&
  hasToolCalls(message) &&
  message?.response_meta?.finish_reason === 'tool_calls'
)

const isToolTraceMessage = (message = {}) => {
  if (message.role === 'tool') return true
  if (message.role !== 'assistant') return false
  const hasVisibleAnswer = hasVisibleMessageText(message)
  return !hasVisibleAnswer && hasToolCalls(message)
}

const getThoughtChainDescription = (descriptionMessage = null) => {
  if (!descriptionMessage) {
    return ''
  }

  return getMessageDisplayContent(descriptionMessage)
}

export const hasToolActivity = (toolTraceRecords = []) => toolTraceRecords.some((record) => {
  const message = record.message || {}
  return (
    record.kind === TOOL_TRACE_KINDS.REASONING ||
    record.kind === TOOL_TRACE_KINDS.RESULT ||
    message.role === 'tool' ||
    hasToolCalls(message)
  )
})

export const shouldRenderToolTrace = (record, toolTraceRecords = []) => {
  if (!record || record.message?.role !== 'assistant') {
    return false
  }

  if (
    record.assistantRenderMode === ASSISTANT_DISPLAY_MODES.TOOL_CHAIN ||
    record.assistantRenderMode === ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN
  ) {
    return true
  }

  return hasToolActivity(toolTraceRecords)
}

const getTraceStatus = (traceRecord = {}) => {
  if (traceRecord.status) {
    return traceRecord.status
  }
  return traceRecord.pending ? TOOL_TRACE_STATUS.LOADING : TOOL_TRACE_STATUS.SUCCESS
}

const buildReasoningItems = (traceRecord) => {
  const status = getTraceStatus(traceRecord)
  const reasoningContent = String(traceRecord.message?.reasoning_content || '').trim()

  return [{
    key: `${traceRecord.key}-reasoning`,
    title: '深度思考',
    description: status === TOOL_TRACE_STATUS.ERROR
      ? '思考中断'
      : (reasoningContent ? '模型正在拆解问题' : '正在分析问题'),
    content: reasoningContent ? (
      <div className="thought-chain-markdown">
        {renderMarkdown(reasoningContent)}
      </div>
    ) : null,
    collapsible: Boolean(reasoningContent),
    status,
    blink: status === TOOL_TRACE_STATUS.LOADING
  }]
}

const buildToolCallItems = ({ record, traceRecord, traceIndex = 0, toolDisplayNames = DEFAULT_TOOL_DISPLAY_NAMES }) => {
  const message = traceRecord.message || {}
  const toolCalls = Array.isArray(message.tool_calls) ? message.tool_calls : []
  const status = getTraceStatus(traceRecord)

  return toolCalls.map((call, index) => {
    const toolName = call.function?.name || call.id || `tool-${index + 1}`
    const formattedArgumentsMarkdown = formatToolArgumentsMarkdown(call.function?.arguments)
    const traceDescription = getThoughtChainDescription(
      traceRecord.traceDescription || (traceIndex === 0 ? record?.planningMessage : null)
    )

    return {
      key: `${traceRecord.key}-call-${call.id || call.index || index}`,
      title: getToolDisplayName(toolName, toolDisplayNames),
      description: status === TOOL_TRACE_STATUS.ERROR
        ? (traceDescription || '工具调用失败')
        : (traceDescription || (formattedArgumentsMarkdown ? '参数已准备' : '等待工具执行')),
      content: formattedArgumentsMarkdown ? (
        <div className="thought-chain-markdown">
          {renderMarkdown(formattedArgumentsMarkdown)}
        </div>
      ) : null,
      collapsible: Boolean(formattedArgumentsMarkdown),
      status,
      blink: status === TOOL_TRACE_STATUS.LOADING
    }
  })
}

const buildToolResultItems = (record, index = 0, toolDisplayNames = DEFAULT_TOOL_DISPLAY_NAMES) => {
  const message = record.message || {}
  if (message.role !== 'tool') {
    return []
  }

  const status = getTraceStatus(record)
  const toolName = message.tool_name || message.name || `tool-result-${index + 1}`
  const content = message.content || ''
  const formattedResultMarkdown = formatToolResultMarkdown(content)

  return [{
    key: `${record.key}-result`,
    title: `${getToolDisplayName(toolName, toolDisplayNames)} 执行结果`,
    description: status === TOOL_TRACE_STATUS.ERROR
      ? '工具执行失败'
      : (content ? '工具返回了可展示内容' : '工具已完成执行'),
    content: formattedResultMarkdown ? (
      <div className="thought-chain-markdown">
        {renderMarkdown(formattedResultMarkdown)}
      </div>
    ) : null,
    collapsible: Boolean(formattedResultMarkdown),
    status,
    blink: status === TOOL_TRACE_STATUS.LOADING
  }]
}

export const buildToolTraceItems = ({ record = null, toolTraceRecords = [], toolDisplayNames = DEFAULT_TOOL_DISPLAY_NAMES }) => {
  return toolTraceRecords.flatMap((traceRecord, traceIndex) => {
    if (!traceRecord?.kind) {
      return []
    }

    if (traceRecord.kind === TOOL_TRACE_KINDS.REASONING) {
      return buildReasoningItems(traceRecord)
    }

    if (traceRecord.kind === TOOL_TRACE_KINDS.CALL) {
      return buildToolCallItems({ record, traceRecord, traceIndex, toolDisplayNames })
    }

    if (traceRecord.kind === TOOL_TRACE_KINDS.RESULT) {
      return buildToolResultItems(traceRecord, traceIndex, toolDisplayNames)
    }

    return []
  }).filter(Boolean)
}

export const buildDisplayMessages = (records) => {
  const display = []

  records.forEach((record) => {
    const message = record.message || {}
    const toolTraceRecords = record.toolTraceRecords || []
    const hasThoughtChain = (
      Array.isArray(toolTraceRecords) &&
      toolTraceRecords.length > 0 &&
      shouldRenderToolTrace(record, toolTraceRecords)
    )

    if (message.role === 'user' || message.role === 'system') {
      display.push({ type: message.role, role: message.role, key: record.key, record })
      return
    }

    if (message.role !== 'assistant') {
      return
    }

    if (hasThoughtChain) {
      display.push({
        type: 'thought_chain',
        role: 'thought_chain',
        key: `${record.key}_thought_chain`,
        record,
        toolTraceRecords
      })
    }

    if (isToolTraceMessage(message)) {
      return
    }

    if (String(message.content || '').trim()) {
      display.push({
        type: 'assistant',
        role: 'assistant',
        key: hasThoughtChain ? `${record.key}_assistant` : record.key,
        record,
        toolTraceRecords
      })
    }
  })

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
