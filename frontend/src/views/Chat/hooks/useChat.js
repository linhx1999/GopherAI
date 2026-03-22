import { useState, useRef, useCallback, useEffect, useMemo } from 'react'
import { App } from 'antd'
import api, { API_BASE_URL } from '../../../utils/api'
import { getStoredToken, handleUnauthorized, isUnauthorizedResponseCode } from '../../../utils/auth'
import useToolCatalog from './useToolCatalog'
import {
  ASSISTANT_DISPLAY_MODES,
  MESSAGE_LIST_BOTTOM_THRESHOLD,
  MESSAGE_PAGE_SIZE,
  STATUS_CODES,
  API_ENDPOINTS,
  MESSAGE_ROLES,
  SPECIAL_SESSIONS,
  SSE_EVENT_TYPES,
  TOOL_TRACE_KINDS,
  TOOL_TRACE_STATUS
} from '../config/constants'
import {
  generateMessageId,
  getAssistantPlainContent,
  hasToolCalls,
  hasVisibleMessageText,
  isToolCallAssistantMessage,
  isSchemaMessagePayload,
  buildDisplayMessages,
  mergeSchemaMessageChunk,
  normalizeEnabledToolAPINames,
  parseSSELine
} from '../utils/helpers.jsx'

const MESSAGE_RENDER_MODE = {
  INSTANT: 'instant',
  STREAM: 'stream'
}

const createEmptyAssistantMessage = () => ({
  role: MESSAGE_ROLES.ASSISTANT,
  content: '',
  reasoning_content: ''
})

const createReasoningStepMessage = (message = {}) => ({
  role: MESSAGE_ROLES.ASSISTANT,
  reasoning_content: message?.reasoning_content || ''
})

const parseSessionItem = (session) => {
  if (!session?.sessionId) {
    return null
  }

  return {
    id: String(session.sessionId),
    title: session.title || `会话 ${session.sessionId}`,
    createdAt: session.createdAt || null
  }
}

const parseSessionListResponse = (responseData) => {
  const sessionList = responseData?.data?.sessions
  if (!Array.isArray(sessionList)) {
    return []
  }

  return sessionList
    .map(parseSessionItem)
    .filter(Boolean)
    .sort((left, right) => new Date(right.createdAt || 0) - new Date(left.createdAt || 0))
}

const parseHistoryResponse = (responseData) => {
  const history = responseData?.data?.messages
  if (!Array.isArray(history)) {
    return []
  }

  const records = []
  let activeAssistantRecord = null

  const pushActiveAssistantRecord = () => {
    if (!activeAssistantRecord) {
      return
    }

    records.push(activeAssistantRecord)
    activeAssistantRecord = null
  }

  history.forEach((item) => {
    const historyMessage = item.message || {}
    const baseMeta = {
      index: item.index,
      createdAt: item.created_at,
      renderMode: MESSAGE_RENDER_MODE.INSTANT
    }

    if (historyMessage.role === MESSAGE_ROLES.USER || historyMessage.role === MESSAGE_ROLES.SYSTEM) {
      pushActiveAssistantRecord()
      records.push(createRecord({
        ...baseMeta,
        key: `msg_${item.index}_${historyMessage.role || 'unknown'}`,
        message: historyMessage,
        pending: false
      }))
      return
    }

    if (isToolCallAssistantMessage(historyMessage)) {
      if (!activeAssistantRecord) {
        activeAssistantRecord = createRecord({
          ...baseMeta,
          key: `msg_${item.index}_assistant_tool_chain`,
          message: createEmptyAssistantMessage(),
          pending: false,
          assistantRenderMode: ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN,
          toolTraceRecords: []
        })
      }

      if (String(historyMessage.reasoning_content || '').trim()) {
        activeAssistantRecord.toolTraceRecords = [
          ...(activeAssistantRecord.toolTraceRecords || []),
          createToolTraceRecord({
            ...baseMeta,
            key: `msg_${item.index}_reasoning`,
            kind: TOOL_TRACE_KINDS.REASONING,
            message: createReasoningStepMessage(historyMessage),
            status: TOOL_TRACE_STATUS.SUCCESS
          })
        ]
      }

      activeAssistantRecord.assistantRenderMode = ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN
      activeAssistantRecord.toolTraceRecords = [
        ...(activeAssistantRecord.toolTraceRecords || []),
        createToolTraceRecord({
          ...baseMeta,
          key: `msg_${item.index}_tool_call`,
          kind: TOOL_TRACE_KINDS.CALL,
          message: historyMessage,
          traceDescription: getAssistantPlainContent(historyMessage)
            ? { role: MESSAGE_ROLES.ASSISTANT, content: getAssistantPlainContent(historyMessage) }
            : null,
          status: TOOL_TRACE_STATUS.SUCCESS
        })
      ]
      return
    }

    if (historyMessage.role === MESSAGE_ROLES.TOOL) {
      if (!activeAssistantRecord) {
        activeAssistantRecord = createRecord({
          ...baseMeta,
          key: `msg_${item.index}_assistant_tool_chain`,
          message: createEmptyAssistantMessage(),
          pending: false,
          assistantRenderMode: ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN,
          toolTraceRecords: []
        })
      }

      activeAssistantRecord.assistantRenderMode = ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN
      activeAssistantRecord.toolTraceRecords = [
        ...(activeAssistantRecord.toolTraceRecords || []),
        createToolTraceRecord({
          ...baseMeta,
          key: `msg_${item.index}_tool_result`,
          kind: TOOL_TRACE_KINDS.RESULT,
          message: historyMessage,
          status: TOOL_TRACE_STATUS.SUCCESS
        })
      ]
      return
    }

    if (historyMessage.role === MESSAGE_ROLES.ASSISTANT) {
      const reasoningTraceRecords = String(historyMessage.reasoning_content || '').trim()
        ? [createToolTraceRecord({
          ...baseMeta,
          key: `msg_${item.index}_reasoning`,
          kind: TOOL_TRACE_KINDS.REASONING,
          message: createReasoningStepMessage(historyMessage),
          status: TOOL_TRACE_STATUS.SUCCESS
        })]
        : []

      if (activeAssistantRecord) {
        activeAssistantRecord.toolTraceRecords = [
          ...(activeAssistantRecord.toolTraceRecords || []),
          ...reasoningTraceRecords
        ]
        activeAssistantRecord.message = historyMessage
        activeAssistantRecord.pending = false
        activeAssistantRecord.assistantRenderMode = activeAssistantRecord.toolTraceRecords.length > 0
          ? ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN
          : ASSISTANT_DISPLAY_MODES.DEFAULT
        pushActiveAssistantRecord()
        return
      }

      records.push(createRecord({
        ...baseMeta,
        key: `msg_${item.index}_assistant`,
        message: historyMessage,
        pending: false,
        assistantRenderMode: reasoningTraceRecords.length > 0
          ? ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN
          : ASSISTANT_DISPLAY_MODES.DEFAULT,
        toolTraceRecords: reasoningTraceRecords
      }))
    }
  })

  pushActiveAssistantRecord()
  return records
}

const createRecord = ({
  key = generateMessageId(),
  index = null,
  message,
  pending = false,
  createdAt = null,
  renderMode = MESSAGE_RENDER_MODE.INSTANT,
  expectReasoning = false,
  assistantRenderMode = ASSISTANT_DISPLAY_MODES.DEFAULT,
  enabledToolAPINames = [],
  planningMessage = null,
  toolTraceRecords = [],
  traceDescription = null,
  kind = null,
  status = null
}) => ({
  key,
  index,
  message,
  pending,
  createdAt,
  renderMode,
  expectReasoning,
  assistantRenderMode,
  enabledToolAPINames,
  planningMessage,
  toolTraceRecords,
  traceDescription,
  kind,
  status
})

const hasMessageIndex = (records = [], messageIndex) => (
  typeof messageIndex === 'number' && records.some((record) => record.index === messageIndex)
)

const hasVisibleAssistantContent = (message = {}) => hasVisibleMessageText(message)

const createToolTraceRecord = ({
  key = generateMessageId(),
  index,
  kind,
  message,
  traceDescription = null,
  status = TOOL_TRACE_STATUS.LOADING,
  renderMode = MESSAGE_RENDER_MODE.STREAM,
  createdAt = null
}) => createRecord({
  key,
  index,
  message: mergeSchemaMessageChunk({}, message),
  pending: status === TOOL_TRACE_STATUS.LOADING,
  renderMode,
  createdAt,
  traceDescription,
  kind,
  status
})

const settleToolTraceRecord = (record, fallbackStatus = TOOL_TRACE_STATUS.SUCCESS) => ({
  ...record,
  pending: false,
  status: record?.status === TOOL_TRACE_STATUS.LOADING ? fallbackStatus : (record?.status || fallbackStatus)
})

const settleRecord = (record, traceFallbackStatus = TOOL_TRACE_STATUS.SUCCESS) => ({
  ...record,
  pending: false,
  toolTraceRecords: Array.isArray(record.toolTraceRecords)
    ? record.toolTraceRecords.map((item) => settleToolTraceRecord(item, traceFallbackStatus))
    : record.toolTraceRecords
})

const useChat = () => {
  const { message } = App.useApp()
  const bubbleListRef = useRef(null)
  const isNearBottomRef = useRef(true)
  const isLastDisplayPageRef = useRef(true)
  const previousLastDisplayPageRef = useRef(1)
  const latestHistorySessionRef = useRef(null)
  const activeKeyRef = useRef(null)
  const isTempSessionRef = useRef(false)
  const { availableTools, availableMCPServers, mcpFeatureEnabled } = useToolCatalog()

  const [enabledToolAPINames, setEnabledToolAPINames] = useState([])
  const [enabledMCPServerIDs, setEnabledMCPServerIDs] = useState([])
  const [thinkingMode, setThinkingMode] = useState(true)
  const [isStreaming, setIsStreaming] = useState(true)
  const [currentPage, setCurrentPage] = useState(1)
  const [inputValue, setInputValue] = useState('')
  const [isLoading, setIsLoading] = useState(false)

  const [attachments, setAttachments] = useState([])
  const [attachmentsOpen, setAttachmentsOpen] = useState(false)

  const [messages, setMessages] = useState([])

  const [sessions, setSessions] = useState([])
  const [activeKey, setActiveKey] = useState(null)
  const [isTempSession, setIsTempSession] = useState(false)
  const [editingSession, setEditingSession] = useState(null)
  const [editTitle, setEditTitle] = useState('')
  const displayMessages = useMemo(() => buildDisplayMessages(messages), [messages])
  const lastDisplayPage = Math.max(1, Math.ceil(displayMessages.length / MESSAGE_PAGE_SIZE))
  const isLastDisplayPage = currentPage >= lastDisplayPage

  useEffect(() => {
    isLastDisplayPageRef.current = isLastDisplayPage
  }, [isLastDisplayPage])

  useEffect(() => {
    const previousLastDisplayPage = previousLastDisplayPageRef.current

    if (
      lastDisplayPage > previousLastDisplayPage &&
      currentPage === previousLastDisplayPage &&
      isNearBottomRef.current
    ) {
      setCurrentPage(lastDisplayPage)
    }

    previousLastDisplayPageRef.current = lastDisplayPage
  }, [currentPage, lastDisplayPage])

  const handleBubbleListScroll = useCallback((event) => {
    const target = event?.target
    if (!target) {
      return
    }

    const distanceToBottom = target.scrollHeight - target.clientHeight - target.scrollTop
    isNearBottomRef.current = distanceToBottom <= MESSAGE_LIST_BOTTOM_THRESHOLD
  }, [])

  const scrollToBottomIfNeeded = useCallback(({ behavior = 'smooth', force = false } = {}) => {
    const listRef = bubbleListRef.current
    if (!listRef?.scrollTo) {
      return
    }

    if (!force && (!isNearBottomRef.current || !isLastDisplayPageRef.current)) {
      return
    }

    requestAnimationFrame(() => {
      bubbleListRef.current?.scrollTo?.({ top: 'bottom', behavior })
    })
  }, [])

  const loadSessions = useCallback(async () => {
    try {
      const response = await api.get(API_ENDPOINTS.SESSIONS)
      if (response.data?.code === STATUS_CODES.SUCCESS) {
        setSessions(parseSessionListResponse(response.data))
      }
    } catch (error) {
      console.error('Load sessions error:', error)
    }
  }, [])

  useEffect(() => {
    loadSessions()
  }, [loadSessions])

  useEffect(() => {
    activeKeyRef.current = activeKey
  }, [activeKey])

  useEffect(() => {
    isTempSessionRef.current = isTempSession
  }, [isTempSession])

  useEffect(() => {
    if (availableTools.length === 0) {
      return
    }

    const availableToolAPINames = new Set(availableTools.map((tool) => tool.apiName))
    setEnabledToolAPINames((previousToolAPINames) => previousToolAPINames.filter((toolName) => availableToolAPINames.has(toolName)))
  }, [availableTools])

  useEffect(() => {
    if (availableMCPServers.length === 0) {
      setEnabledMCPServerIDs([])
      return
    }

    const availableServerIDs = new Set(availableMCPServers.map((server) => server.serverId))
    setEnabledMCPServerIDs((previousServerIDs) => previousServerIDs.filter((serverId) => availableServerIDs.has(serverId)))
  }, [availableMCPServers])

  const loadMessages = useCallback(async (sessionId) => {
    const targetSessionId = String(sessionId)
    latestHistorySessionRef.current = targetSessionId

    try {
      const response = await api.get(API_ENDPOINTS.AGENT_MESSAGES(targetSessionId))
      if (latestHistorySessionRef.current !== targetSessionId) {
        return
      }

      if (response.data?.code === STATUS_CODES.SUCCESS) {
        const historyRecords = parseHistoryResponse(response.data)
        setMessages(historyRecords)
        return historyRecords
      }

      setMessages([])
      if (response.data?.msg) {
        message.error(response.data.msg)
      }
      return []
    } catch (err) {
      if (latestHistorySessionRef.current !== targetSessionId) {
        return []
      }
      console.error('Load history error:', err)
      setMessages([])
      return []
    }
  }, [message])

  const createSession = useCallback(() => {
    const createdAt = new Date().toISOString()
    latestHistorySessionRef.current = SPECIAL_SESSIONS.TEMP
    setSessions((prev) => {
      const nextSessions = prev.filter((session) => session.id !== SPECIAL_SESSIONS.TEMP)
      return [{ id: SPECIAL_SESSIONS.TEMP, title: '新会话', createdAt }, ...nextSessions]
    })
    setActiveKey(SPECIAL_SESSIONS.TEMP)
    setIsTempSession(true)
    setMessages([])
    setCurrentPage(1)
  }, [])

  const switchSession = useCallback(async (sessionId) => {
    if (!sessionId) return

    setActiveKey(sessionId)
    setCurrentPage(1)
    if (sessionId === SPECIAL_SESSIONS.TEMP) {
      latestHistorySessionRef.current = SPECIAL_SESSIONS.TEMP
      setIsTempSession(true)
      setMessages([])
      return
    }

    setIsTempSession(false)
    await loadMessages(sessionId)
  }, [loadMessages])

  const deleteSession = useCallback(async (sessionId) => {
    try {
      const response = await api.delete(API_ENDPOINTS.SESSION_DELETE(sessionId))
      if (response.data?.code === STATUS_CODES.SUCCESS) {
        message.success('会话已删除')
        setSessions((prev) => prev.filter((s) => s.id !== sessionId))
        if (activeKey === sessionId) {
          setActiveKey(null)
          setIsTempSession(false)
          setMessages([])
        }
      } else {
        message.error(response.data?.msg || '删除失败')
      }
    } catch (error) {
      console.error('Delete session error:', error)
      message.error('删除会话失败')
    }
  }, [activeKey, message])

  const updateSessionTitle = useCallback(async (sessionId, title) => {
    try {
      const response = await api.put(API_ENDPOINTS.SESSION_TITLE(sessionId), { title })
      if (response.data?.code === STATUS_CODES.SUCCESS) {
        message.success('标题已更新')
        setSessions((prev) => prev.map((s) => (s.id === sessionId ? { ...s, title } : s)))
      } else {
        message.error(response.data?.msg || '更新失败')
      }
    } catch (error) {
      console.error('Update title error:', error)
      message.error('更新标题失败')
    }
  }, [message])

  const shareSession = useCallback(async (sessionId) => {
    try {
      const shareUrl = `${window.location.origin}/shared-chat/${sessionId}`
      await navigator.clipboard.writeText(shareUrl)
      message.success('分享链接已复制到剪贴板')
    } catch (error) {
      console.error('Share session error:', error)
      message.error('复制链接失败')
    }
  }, [message])

  const confirmRename = useCallback(async () => {
    if (!editingSession || !editTitle.trim()) {
      setEditingSession(null)
      return
    }
    await updateSessionTitle(editingSession, editTitle.trim())
    setEditingSession(null)
    setEditTitle('')
  }, [editingSession, editTitle, updateSessionTitle])

  const handleMenuClick = useCallback((itemInfo, sessionId) => {
    itemInfo.domEvent.stopPropagation()
    const { key } = itemInfo

    switch (key) {
      case 'rename': {
        const session = sessions.find((s) => s.id === sessionId)
        if (session) {
          setEditingSession(sessionId)
          setEditTitle(session.title || '')
        }
        break
      }
      case 'share':
        shareSession(sessionId)
        break
      case 'deleteChat':
        deleteSession(sessionId)
        break
    }
  }, [sessions, shareSession, deleteSession])

  const bindServerSession = useCallback((sessionId, sessionPatch = {}) => {
    const newSid = String(sessionId)
    const shouldBindActiveSession = !activeKeyRef.current || isTempSessionRef.current
    const nextSession = {
      id: newSid,
      title: sessionPatch.title || `会话 ${newSid}`,
      createdAt: sessionPatch.createdAt || new Date().toISOString()
    }

    setSessions((prev) => {
      const nextSessions = prev.map((session) => (
        shouldBindActiveSession && session.id === SPECIAL_SESSIONS.TEMP
          ? { ...session, ...nextSession }
          : session
      ))

      const existingIndex = nextSessions.findIndex((session) => session.id === newSid)
      if (existingIndex >= 0) {
        if (!sessionPatch.title && !sessionPatch.createdAt) {
          return nextSessions
        }

        const mergedSessions = [...nextSessions]
        mergedSessions[existingIndex] = {
          ...mergedSessions[existingIndex],
          ...sessionPatch
        }
        return mergedSessions
      }

      if (!shouldBindActiveSession) {
        return prev
      }

      return [nextSession, ...nextSessions]
    })

    if (shouldBindActiveSession) {
      setActiveKey(newSid)
      setIsTempSession(false)
    }

    latestHistorySessionRef.current = newSid
  }, [])

  const createServerSession = useCallback(async () => {
    const response = await api.post(API_ENDPOINTS.SESSIONS)
    if (response.data?.code !== STATUS_CODES.SUCCESS) {
      throw new Error(response.data?.msg || '创建会话失败')
    }

    const createdSession = parseSessionItem(response.data?.data?.session)
    if (!createdSession) {
      throw new Error('创建会话失败')
    }

    bindServerSession(createdSession.id, createdSession)
    return createdSession
  }, [bindServerSession])

  const buildChatRunOptions = useCallback(() => ({
    enabledToolAPINames: normalizeEnabledToolAPINames(enabledToolAPINames),
    enabledMCPServerIDs: normalizeEnabledToolAPINames(enabledMCPServerIDs),
    thinkingModeEnabled: thinkingMode
  }), [enabledMCPServerIDs, enabledToolAPINames, thinkingMode])

  const buildChatRequest = useCallback((question, runOptions, sessionIdOverride = null) => {
    const payload = {
      message: question,
      tools: runOptions.enabledToolAPINames,
      mcp_server_ids: runOptions.enabledMCPServerIDs,
      thinking_mode: runOptions.thinkingModeEnabled,
    }

    const targetSessionId = sessionIdOverride ?? ((activeKey && !isTempSession) ? activeKey : null)
    if (targetSessionId) {
      payload.session_id = targetSessionId
    }
    return payload
  }, [activeKey, isTempSession])

  const sendStreamMessage = useCallback(async (question, runOptions) => {
    const url = `${API_BASE_URL}/${API_ENDPOINTS.AGENT_STREAM}`
    let didCreateSession = false
    let activeMessageKey = null
    let activeRecordMode = ASSISTANT_DISPLAY_MODES.DEFAULT
    let activeAssistantMessage = createEmptyAssistantMessage()
    let activeReasoningTraceKey = null
    let activeToolCallTraceKey = null
    let activeToolCallMessage = null
    let activeToolResultTraceKey = null
    let activeToolResultMessage = null
    const createdRecordKeys = []

    const finalizeStream = (traceFallbackStatus = TOOL_TRACE_STATUS.SUCCESS) => {
      if (createdRecordKeys.length > 0) {
        setMessages((prev) => prev.map((item) => (
          createdRecordKeys.includes(item.key) ? settleRecord(item, traceFallbackStatus) : item
        )))
      }
      activeMessageKey = null
      activeAssistantMessage = createEmptyAssistantMessage()
      activeRecordMode = ASSISTANT_DISPLAY_MODES.DEFAULT
      activeReasoningTraceKey = null
      activeToolCallTraceKey = null
      activeToolCallMessage = null
      activeToolResultTraceKey = null
      activeToolResultMessage = null
      setIsLoading(false)
    }

    const ensureActiveAssistantRecord = (assistantRenderMode = ASSISTANT_DISPLAY_MODES.DEFAULT) => {
      if (activeMessageKey) {
        return
      }

      const record = createRecord({
        index: null,
        message: createEmptyAssistantMessage(),
        pending: true,
        renderMode: MESSAGE_RENDER_MODE.STREAM,
        expectReasoning: runOptions.thinkingModeEnabled,
        assistantRenderMode,
        enabledToolAPINames: runOptions.enabledToolAPINames,
        toolTraceRecords: []
      })

      activeMessageKey = record.key
      activeRecordMode = assistantRenderMode
      activeAssistantMessage = createEmptyAssistantMessage()
      createdRecordKeys.push(record.key)
      setMessages((prev) => [...prev, record])
    }

    const patchActiveRecord = (updater) => {
      if (!activeMessageKey) {
        return
      }

      setMessages((prev) => prev.map((item) => {
        if (item.key !== activeMessageKey) {
          return item
        }

        return updater(item)
      }))
    }

    const setActiveAssistantState = ({
      message: nextMessage = activeAssistantMessage,
      pending = true,
      assistantRenderMode = activeRecordMode
    } = {}) => {
      ensureActiveAssistantRecord(assistantRenderMode)
      activeAssistantMessage = nextMessage
      activeRecordMode = assistantRenderMode

      patchActiveRecord((item) => ({
        ...item,
        message: nextMessage,
        pending,
        assistantRenderMode
      }))
    }

    const appendToolTraceRecord = (traceRecord) => {
      ensureActiveAssistantRecord(ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN)
      activeRecordMode = ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN

      patchActiveRecord((item) => ({
        ...item,
        message: item.message || createEmptyAssistantMessage(),
        pending: true,
        assistantRenderMode: ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN,
        toolTraceRecords: [...(item.toolTraceRecords || []), traceRecord]
      }))
    }

    const updateToolTraceRecord = (traceKey, updater) => {
      if (!traceKey) {
        return
      }

      patchActiveRecord((item) => {
        const toolTraceRecords = Array.isArray(item.toolTraceRecords) ? [...item.toolTraceRecords] : []
        const traceIndex = toolTraceRecords.findIndex((record) => record.key === traceKey)
        if (traceIndex === -1) {
          return item
        }

        toolTraceRecords[traceIndex] = updater(toolTraceRecords[traceIndex])
        return {
          ...item,
          toolTraceRecords
        }
      })
    }

    const removeToolTraceRecord = (traceKey) => {
      if (!traceKey) {
        return
      }

      patchActiveRecord((item) => ({
        ...item,
        toolTraceRecords: (item.toolTraceRecords || []).filter((traceRecord) => traceRecord.key !== traceKey)
      }))
    }

    const ensureReasoningStep = ({ placeholder = false } = {}) => {
      ensureActiveAssistantRecord(ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN)

      const reasoningMessage = createReasoningStepMessage(activeAssistantMessage)
      const hasReasoning = Boolean(String(reasoningMessage.reasoning_content || '').trim())
      if (!placeholder && !hasReasoning) {
        return
      }

      if (!activeReasoningTraceKey) {
        const traceRecord = createToolTraceRecord({
          kind: TOOL_TRACE_KINDS.REASONING,
          message: reasoningMessage,
          status: TOOL_TRACE_STATUS.LOADING
        })

        activeReasoningTraceKey = traceRecord.key
        appendToolTraceRecord(traceRecord)
        return
      }

      updateToolTraceRecord(activeReasoningTraceKey, (traceRecord) => ({
        ...traceRecord,
        message: reasoningMessage,
        pending: true,
        status: TOOL_TRACE_STATUS.LOADING
      }))
    }

    const finalizeReasoningStep = (messagePayload, status = TOOL_TRACE_STATUS.SUCCESS) => {
      if (!activeReasoningTraceKey) {
        return
      }

      const reasoningMessage = createReasoningStepMessage(
        mergeSchemaMessageChunk(activeAssistantMessage, messagePayload)
      )
      const hasReasoning = Boolean(String(reasoningMessage.reasoning_content || '').trim())

      if (!hasReasoning && status === TOOL_TRACE_STATUS.SUCCESS) {
        removeToolTraceRecord(activeReasoningTraceKey)
        activeReasoningTraceKey = null
        return
      }

      updateToolTraceRecord(activeReasoningTraceKey, (traceRecord) => ({
        ...traceRecord,
        message: reasoningMessage,
        pending: status === TOOL_TRACE_STATUS.LOADING,
        status
      }))

      if (status !== TOOL_TRACE_STATUS.LOADING) {
        activeReasoningTraceKey = null
      }
    }

    const startOrUpdateToolCallStep = (chunk) => {
      ensureActiveAssistantRecord(ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN)
      activeToolCallMessage = mergeSchemaMessageChunk(
        activeToolCallMessage || createEmptyAssistantMessage(),
        {
          ...chunk,
          content: '',
          reasoning_content: ''
        }
      )

      if (!activeToolCallTraceKey) {
        const plainContent = getAssistantPlainContent(activeAssistantMessage)
        const traceDescription = plainContent
          ? { role: MESSAGE_ROLES.ASSISTANT, content: plainContent }
          : null

        const traceRecord = createToolTraceRecord({
          kind: TOOL_TRACE_KINDS.CALL,
          message: activeToolCallMessage,
          traceDescription,
          status: TOOL_TRACE_STATUS.LOADING
        })

        activeToolCallTraceKey = traceRecord.key
        activeAssistantMessage = createEmptyAssistantMessage()
        activeRecordMode = ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN

        patchActiveRecord((item) => ({
          ...item,
          message: createEmptyAssistantMessage(),
          pending: true,
          assistantRenderMode: ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN,
          toolTraceRecords: [...(item.toolTraceRecords || []), traceRecord]
        }))
        return
      }

      updateToolTraceRecord(activeToolCallTraceKey, (traceRecord) => ({
        ...traceRecord,
        message: activeToolCallMessage,
        pending: true,
        status: TOOL_TRACE_STATUS.LOADING
      }))
    }

    const completeToolCallStep = (messagePayload) => {
      ensureActiveAssistantRecord(ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN)
      finalizeReasoningStep(messagePayload)
      const completedMessage = mergeSchemaMessageChunk(
        activeToolCallMessage || createEmptyAssistantMessage(),
        messagePayload
      )

      const plainContent = getAssistantPlainContent(messagePayload)
      if (activeToolCallTraceKey) {
        updateToolTraceRecord(activeToolCallTraceKey, (traceRecord) => ({
          ...traceRecord,
          message: completedMessage,
          pending: false,
          status: TOOL_TRACE_STATUS.SUCCESS,
          traceDescription: traceRecord.traceDescription || (
            plainContent
              ? { role: MESSAGE_ROLES.ASSISTANT, content: plainContent }
              : null
          )
        }))
      } else {
        appendToolTraceRecord(createToolTraceRecord({
          kind: TOOL_TRACE_KINDS.CALL,
          message: completedMessage,
          traceDescription: plainContent
            ? { role: MESSAGE_ROLES.ASSISTANT, content: plainContent }
            : null,
          status: TOOL_TRACE_STATUS.SUCCESS
        }))
      }

      activeToolCallTraceKey = null
      activeToolCallMessage = null
      activeAssistantMessage = createEmptyAssistantMessage()
      activeRecordMode = ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN

      patchActiveRecord((item) => ({
        ...item,
        message: createEmptyAssistantMessage(),
        pending: true,
        assistantRenderMode: ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN
      }))
    }

    const startOrUpdateToolResultStep = (chunk) => {
      ensureActiveAssistantRecord(ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN)
      activeToolResultMessage = mergeSchemaMessageChunk(activeToolResultMessage || {}, chunk)

      if (!activeToolResultTraceKey) {
        const traceRecord = createToolTraceRecord({
          kind: TOOL_TRACE_KINDS.RESULT,
          message: activeToolResultMessage,
          status: TOOL_TRACE_STATUS.LOADING
        })

        activeToolResultTraceKey = traceRecord.key
        appendToolTraceRecord(traceRecord)
        return
      }

      updateToolTraceRecord(activeToolResultTraceKey, (traceRecord) => ({
        ...traceRecord,
        message: activeToolResultMessage,
        pending: true,
        status: TOOL_TRACE_STATUS.LOADING
      }))
    }

    const completeToolResultStep = (messagePayload) => {
      ensureActiveAssistantRecord(ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN)
      const completedMessage = mergeSchemaMessageChunk(activeToolResultMessage || {}, messagePayload)

      if (activeToolResultTraceKey) {
        updateToolTraceRecord(activeToolResultTraceKey, (traceRecord) => ({
          ...traceRecord,
          message: completedMessage,
          pending: false,
          status: TOOL_TRACE_STATUS.SUCCESS
        }))
      } else {
        appendToolTraceRecord(createToolTraceRecord({
          kind: TOOL_TRACE_KINDS.RESULT,
          message: completedMessage,
          status: TOOL_TRACE_STATUS.SUCCESS
        }))
      }

      activeToolResultTraceKey = null
      activeToolResultMessage = null
      activeRecordMode = ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN
    }

    const updateAssistantDelta = (chunk) => {
      ensureActiveAssistantRecord(activeRecordMode)
      activeAssistantMessage = mergeSchemaMessageChunk(activeAssistantMessage, chunk)
      if (String(activeAssistantMessage.reasoning_content || '').trim()) {
        ensureReasoningStep()
      }
      setActiveAssistantState({
        message: activeAssistantMessage,
        pending: true,
        assistantRenderMode: activeRecordMode
      })
    }

    const completeAssistantMessage = (messagePayload) => {
      ensureActiveAssistantRecord(activeRecordMode)
      finalizeReasoningStep(messagePayload)
      const completedMessage = mergeSchemaMessageChunk(activeAssistantMessage, messagePayload)
      const nextRenderMode = activeRecordMode === ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN
        ? ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN
        : ASSISTANT_DISPLAY_MODES.DEFAULT

      patchActiveRecord((item) => ({
        ...item,
        message: completedMessage,
        pending: false,
        assistantRenderMode: (item.toolTraceRecords || []).length > 0 ? ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN : nextRenderMode
      }))

      activeMessageKey = null
      activeAssistantMessage = createEmptyAssistantMessage()
      activeRecordMode = ASSISTANT_DISPLAY_MODES.DEFAULT
      activeReasoningTraceKey = null
      activeToolCallTraceKey = null
      activeToolCallMessage = null
      activeToolResultTraceKey = null
      activeToolResultMessage = null
    }

    const markStreamFailed = () => {
      finalizeReasoningStep(activeAssistantMessage, TOOL_TRACE_STATUS.ERROR)
      finalizeStream(TOOL_TRACE_STATUS.ERROR)
    }

    try {
      let sessionId = activeKey && !isTempSession ? activeKey : null
      if (!sessionId) {
        const createdSession = await createServerSession()
        sessionId = createdSession.id
        didCreateSession = true
      }

      const payload = buildChatRequest(question, runOptions, sessionId)
      const response = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${getStoredToken() || ''}`
        },
        body: JSON.stringify(payload)
      })

      if (response.status === 401) {
        finalizeStream()
        handleUnauthorized()
        return
      }

      const responseContentType = response.headers.get('content-type') || ''
      if (responseContentType.includes('application/json')) {
        const responseData = await response.json()
        if (isUnauthorizedResponseCode(responseData?.code)) {
          finalizeStream()
          handleUnauthorized()
          return
        }

        throw new Error(responseData?.msg || `HTTP error! status: ${response.status}`)
      }

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          const parsed = parseSSELine(line)
          if (!parsed) continue

          if (parsed.type !== 'json') {
            continue
          }

          const event = parsed.data
          if (event?.type === SSE_EVENT_TYPES.RESPONSE_CREATED) {
            if (runOptions.thinkingModeEnabled) {
              ensureActiveAssistantRecord(ASSISTANT_DISPLAY_MODES.THOUGHT_CHAIN)
              ensureReasoningStep({ placeholder: true })
            }
            continue
          }

          if (event?.type === SSE_EVENT_TYPES.RESPONSE_DONE) {
            finalizeStream()
            continue
          }

          if (event?.type === SSE_EVENT_TYPES.RESPONSE_MESSAGE_COMPLETED) {
            const payload = event?.response?.message
            if (!isSchemaMessagePayload(payload)) {
              continue
            }

            if (payload.role === MESSAGE_ROLES.ASSISTANT) {
              if (isToolCallAssistantMessage(payload)) {
                completeToolCallStep(payload)
                continue
              }

              completeAssistantMessage(payload)
              continue
            }

            if (payload.role === MESSAGE_ROLES.TOOL) {
              completeToolResultStep(payload)
            }
            continue
          }

          if (event?.type === SSE_EVENT_TYPES.RESPONSE_ERROR) {
            markStreamFailed()
            message.error(event.message || '流式传输出错')
            return
          }

          if (event?.type !== SSE_EVENT_TYPES.RESPONSE_MESSAGE_DELTA) {
            continue
          }

          const payload = event?.response?.delta

          if (!isSchemaMessagePayload(payload)) {
            continue
          }

          if (payload.role === MESSAGE_ROLES.ASSISTANT) {
            if (hasToolCalls(payload)) {
              startOrUpdateToolCallStep(payload)
              continue
            }

            if (!hasVisibleAssistantContent(payload)) {
              continue
            }

            updateAssistantDelta(payload)
            continue
          }

          if (payload.role === MESSAGE_ROLES.TOOL) {
            startOrUpdateToolResultStep(payload)
          }
        }
      }

      finalizeStream()
    } catch (err) {
      console.error('Stream error:', err)
      markStreamFailed()
      message.error('流式传输出错: ' + err.message)
    } finally {
      if (didCreateSession) {
        loadSessions()
      }
    }
  }, [activeKey, buildChatRequest, createServerSession, isTempSession, loadSessions, message])

  const sendGenerateMessage = useCallback(async (question, runOptions) => {
    const payload = buildChatRequest(question, runOptions)
    const hasEnabledTools = runOptions.enabledToolAPINames.length > 0 || runOptions.enabledMCPServerIDs.length > 0
    try {
      const response = await api.post(API_ENDPOINTS.AGENT_GENERATE, payload)

      if (response.data?.code === STATUS_CODES.SUCCESS) {
        const data = response.data?.data || {}
        let historyRecords = []
        if (data.session_id) {
          bindServerSession(String(data.session_id))
          historyRecords = await loadMessages(String(data.session_id))
        } else if (activeKey && !isTempSession) {
          historyRecords = await loadMessages(activeKey)
        }

        const shouldAppendFallback = (
          !hasEnabledTools &&
          typeof data.message_index === 'number' &&
          isSchemaMessagePayload(data.message) &&
          !hasMessageIndex(historyRecords, data.message_index)
        )

        if (shouldAppendFallback) {
          setMessages((prev) => {
            if (hasMessageIndex(prev, data.message_index)) {
              return prev
            }

            return [...prev, createRecord({
              index: data.message_index,
              message: data.message,
              pending: false,
              renderMode: MESSAGE_RENDER_MODE.INSTANT,
              expectReasoning: runOptions.thinkingModeEnabled && data.message.role === MESSAGE_ROLES.ASSISTANT,
              assistantRenderMode: ASSISTANT_DISPLAY_MODES.DEFAULT,
              enabledToolAPINames: runOptions.enabledToolAPINames
            })]
          })
        }
      } else {
        message.error(response.data?.msg || '发送失败')
      }
    } catch (err) {
      console.error('Send message error:', err)
      message.error('发送失败，请重试')
    } finally {
      setIsLoading(false)
    }
  }, [activeKey, bindServerSession, buildChatRequest, isTempSession, loadMessages, message])

  const handleAttachmentUpload = useCallback(async (file) => {
    const formData = new FormData()
    formData.append('file', file)
    try {
      const response = await api.post(API_ENDPOINTS.FILE_UPLOAD, formData, {
        headers: { 'Content-Type': 'multipart/form-data' }
      })
      if (response.data?.code === STATUS_CODES.SUCCESS) {
        message.success(`${file.name} 上传成功`)
        return response.data.data?.[0]?.id || null
      }
      message.error(response.data?.msg || '上传失败')
      return null
    } catch (error) {
      console.error('Attachment upload error:', error)
      message.error(`${file.name} 上传失败`)
      return null
    }
  }, [message])

  const handleSend = useCallback(async (content) => {
    if (!content.trim() && attachments.length === 0) {
      message.warning('请输入消息内容或添加附件')
      return
    }

    setIsLoading(true)

    let uploadedFileIds = []
    if (attachments.length > 0) {
      const uploadPromises = attachments
        .filter((item) => item.originFileObj)
        .map((item) => handleAttachmentUpload(item.originFileObj))
      const results = await Promise.all(uploadPromises)
      uploadedFileIds = results.filter((id) => id !== null)
    }

    let messageContent = content
    if (uploadedFileIds.length > 0) {
      const fileNames = attachments.map((item) => item.name).join(', ')
      messageContent = content ? `${content}\n\n[附件: ${fileNames}]` : `[附件: ${fileNames}]`
    }

    const userRecord = createRecord({
      message: { role: MESSAGE_ROLES.USER, content: messageContent },
      renderMode: MESSAGE_RENDER_MODE.INSTANT
    })
    setMessages((prev) => [...prev, userRecord])

    setInputValue('')
    attachments.forEach((item) => {
      if (item.url?.startsWith('blob:')) {
        URL.revokeObjectURL(item.url)
      }
    })
    setAttachments([])
    setAttachmentsOpen(false)

    setCurrentPage((prevPage) => {
      const total = displayMessages.length + 1
      const page = Math.ceil(total / MESSAGE_PAGE_SIZE)
      return prevPage !== page ? page : prevPage
    })

    const runOptions = buildChatRunOptions()

    if (isStreaming) {
      await sendStreamMessage(messageContent, runOptions)
    } else {
      await sendGenerateMessage(messageContent, runOptions)
    }
  }, [attachments, buildChatRunOptions, displayMessages.length, handleAttachmentUpload, isStreaming, message, sendGenerateMessage, sendStreamMessage])

  const playTTS = useCallback(async (text) => {
    try {
      const response = await api.post(API_ENDPOINTS.TTS, { text })
      if (response.data?.code === STATUS_CODES.SUCCESS && response.data.data?.[0]?.task_id) {
        const taskId = response.data.data[0].task_id
        let attempts = 0
        while (attempts < 30) {
          const queryResponse = await api.get(API_ENDPOINTS.TTS_QUERY, { params: { task_id: taskId } })
          if (queryResponse.data?.code === STATUS_CODES.SUCCESS) {
            const ttsData = queryResponse.data.data?.[0] || {}
            if (ttsData.task_status === 'Success' && ttsData.task_result) {
              const audio = new Audio(ttsData.task_result)
              audio.play()
              return
            }
            if (ttsData.task_status === 'Failed') {
              message.error('语音合成失败')
              return
            }
          }
          attempts += 1
          await new Promise((resolve) => setTimeout(resolve, 2000))
        }
        message.error('语音合成超时')
      } else {
        message.error('无法创建语音合成任务')
      }
    } catch (error) {
      console.error('TTS error:', error)
      message.error('请求语音接口失败')
    }
  }, [message])

  const handleActionClick = useCallback((record, { key }) => {
    switch (key) {
      case 'copy':
        navigator.clipboard.writeText(record.message?.content || '')
        break
      case 'tts':
        playTTS(record.message?.content || '')
        break
      default:
        break
    }
  }, [playTTS])

  useEffect(() => {
    return () => {
      attachments.forEach((item) => {
        if (item.url?.startsWith('blob:')) {
          URL.revokeObjectURL(item.url)
        }
      })
    }
  }, [attachments])

  useEffect(() => {
    scrollToBottomIfNeeded({ behavior: 'smooth' })
  }, [currentPage, displayMessages.length, messages, scrollToBottomIfNeeded])

  return {
    bubbleListRef,
    handleBubbleListScroll,
    availableTools,
    availableMCPServers,
    mcpFeatureEnabled,
    enabledToolAPINames,
    enabledMCPServerIDs,
    thinkingMode,
    isStreaming,
    currentPage,
    inputValue,
    isLoading,
    attachments,
    attachmentsOpen,
    messages,
    sessions,
    activeKey,
    isTempSession,
    editingSession,
    editTitle,
    createSession,
    switchSession,
    handleMenuClick,
    confirmRename,
    setEditTitle,
    handleSend,
    handleActionClick,
    setEnabledToolAPINames,
    setEnabledMCPServerIDs,
    setThinkingMode,
    setIsStreaming,
    setCurrentPage,
    setInputValue,
    setAttachments,
    setAttachmentsOpen
  }
}

export default useChat
