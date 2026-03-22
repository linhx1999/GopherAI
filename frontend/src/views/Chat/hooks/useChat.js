import { useState, useRef, useCallback, useEffect } from 'react'
import { App } from 'antd'
import api, { API_BASE_URL } from '../../../utils/api'
import { getStoredToken, handleUnauthorized, isUnauthorizedResponseCode } from '../../../utils/auth'
import useToolCatalog from './useToolCatalog'
import {
  ASSISTANT_DISPLAY_MODES,
  MESSAGE_PAGE_SIZE,
  STATUS_CODES,
  API_ENDPOINTS,
  MESSAGE_ROLES,
  SPECIAL_SESSIONS,
  SSE_EVENT_TYPES
} from '../config/constants'
import {
  generateMessageId,
  isMessageFinished,
  isSchemaMessagePayload,
  mergeSchemaMessageChunk,
  normalizeEnabledToolAPINames,
  parseSSELine
} from '../utils/helpers.jsx'

const MESSAGE_RENDER_MODE = {
  INSTANT: 'instant',
  STREAM: 'stream'
}

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

  return history.map((item) => createRecord({
    key: `msg_${item.index}_${item.message?.role || 'unknown'}`,
    index: item.index,
    message: item.message || {},
    pending: false,
    createdAt: item.created_at,
    renderMode: MESSAGE_RENDER_MODE.INSTANT
  }))
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
  traceDescription = null
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
  traceDescription
})

const hasMessageIndex = (records = [], messageIndex) => (
  typeof messageIndex === 'number' && records.some((record) => record.index === messageIndex)
)

const hasVisibleAssistantContent = (message = {}) => Boolean(
  String(message?.content || '').trim() || String(message?.reasoning_content || '').trim()
)

const createToolTraceRecord = ({
  index,
  chunk,
  traceDescription = null,
  pending = chunk?.role === MESSAGE_ROLES.TOOL ? false : !isMessageFinished(chunk)
}) => createRecord({
  index,
  message: mergeSchemaMessageChunk({}, chunk),
  pending,
  renderMode: MESSAGE_RENDER_MODE.STREAM,
  traceDescription
})

const settleRecord = (record) => ({
  ...record,
  pending: false,
  toolTraceRecords: Array.isArray(record.toolTraceRecords)
    ? record.toolTraceRecords.map((item) => ({ ...item, pending: false }))
    : record.toolTraceRecords
})

const useChat = () => {
  const { message } = App.useApp()
  const bubbleListRef = useRef(null)
  const latestHistorySessionRef = useRef(null)
  const activeKeyRef = useRef(null)
  const isTempSessionRef = useRef(false)
  const { availableTools } = useToolCatalog()

  const [enabledToolAPINames, setEnabledToolAPINames] = useState([])
  const [thinkingMode, setThinkingMode] = useState(false)
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
    thinkingModeEnabled: thinkingMode
  }), [enabledToolAPINames, thinkingMode])

  const buildChatRequest = useCallback((question, runOptions, sessionIdOverride = null) => {
    const payload = {
      message: question,
      tools: runOptions.enabledToolAPINames,
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
    let nextMessageIndex = null
    let activeMessageKey = null
    let activeRecordMode = ASSISTANT_DISPLAY_MODES.DEFAULT
    let activeToolTraceKey = null
    let pendingToolCallChunk = null
    const createdRecordKeys = []

    const finalizeStream = () => {
      if (createdRecordKeys.length > 0) {
        setMessages((prev) => prev.map((item) => (
          createdRecordKeys.includes(item.key) ? settleRecord(item) : item
        )))
      }
      activeMessageKey = null
      activeRecordMode = ASSISTANT_DISPLAY_MODES.DEFAULT
      activeToolTraceKey = null
      pendingToolCallChunk = null
      setIsLoading(false)
    }

    const createAssistantRecord = (chunk) => {
      const record = createRecord({
        index: nextMessageIndex,
        message: mergeSchemaMessageChunk({}, chunk),
        pending: !isMessageFinished(chunk),
        renderMode: MESSAGE_RENDER_MODE.STREAM,
        expectReasoning: runOptions.thinkingModeEnabled && chunk.role === MESSAGE_ROLES.ASSISTANT,
        assistantRenderMode: ASSISTANT_DISPLAY_MODES.DEFAULT,
        enabledToolAPINames: runOptions.enabledToolAPINames
      })
      if (nextMessageIndex !== null) {
        nextMessageIndex += 1
      }
      activeMessageKey = isMessageFinished(record.message) ? null : record.key
      activeRecordMode = ASSISTANT_DISPLAY_MODES.DEFAULT
      activeToolTraceKey = null
      pendingToolCallChunk = null
      createdRecordKeys.push(record.key)
      setMessages((prev) => [...prev, record])
    }

    const updateActiveAssistantRecord = (chunk) => {
      setMessages((prev) => prev.map((item) => {
        if (item.key !== activeMessageKey) {
          return item
        }
        const mergedMessage = mergeSchemaMessageChunk(item.message, chunk)
        const pending = !isMessageFinished(mergedMessage)
        if (!pending) {
          activeMessageKey = null
          activeRecordMode = ASSISTANT_DISPLAY_MODES.DEFAULT
          activeToolTraceKey = null
          pendingToolCallChunk = null
        }
        return {
          ...item,
          message: mergedMessage,
          pending
        }
      }))
    }

    const appendToolTraceToActiveRecord = (chunk, { mergeIntoLast = false, traceDescription = null } = {}) => {
      setMessages((prev) => prev.map((item) => {
        if (item.key !== activeMessageKey) {
          return item
        }

        const toolTraceRecords = Array.isArray(item.toolTraceRecords) ? [...item.toolTraceRecords] : []
        if (mergeIntoLast && activeToolTraceKey) {
          const traceIndex = toolTraceRecords.findIndex((record) => record.key === activeToolTraceKey)
          if (traceIndex >= 0) {
            const traceRecord = toolTraceRecords[traceIndex]
            const mergedMessage = mergeSchemaMessageChunk(traceRecord.message, chunk)
            const pending = traceRecord.message?.role === MESSAGE_ROLES.TOOL ? false : !isMessageFinished(mergedMessage)
            toolTraceRecords[traceIndex] = {
              ...traceRecord,
              message: mergedMessage,
              pending
            }
            return { ...item, toolTraceRecords }
          }
        }

        const traceRecord = createToolTraceRecord({
          index: null,
          chunk,
          traceDescription
        })
        activeToolTraceKey = traceRecord.key
        return {
          ...item,
          toolTraceRecords: [...toolTraceRecords, traceRecord]
        }
      }))
    }

    const commitPendingToolCallToThoughtChain = () => {
      if (!pendingToolCallChunk) {
        return
      }

      setMessages((prev) => prev.map((item) => {
        if (item.key !== activeMessageKey) {
          return item
        }

        const traceDescription = hasVisibleAssistantContent(item.message)
          ? mergeSchemaMessageChunk({}, item.message)
          : null

        const traceRecord = createToolTraceRecord({
          index: null,
          chunk: pendingToolCallChunk,
          traceDescription,
          pending: true
        })

        activeRecordMode = ASSISTANT_DISPLAY_MODES.TOOL_CHAIN
        activeToolTraceKey = traceRecord.key

        return {
          ...item,
          message: { role: MESSAGE_ROLES.ASSISTANT, content: '', reasoning_content: '' },
          toolTraceRecords: [...(item.toolTraceRecords || []), traceRecord],
          assistantRenderMode: ASSISTANT_DISPLAY_MODES.TOOL_CHAIN,
          enabledToolAPINames: runOptions.enabledToolAPINames,
          pending: true
        }
      }))

      pendingToolCallChunk = null
    }

    const createToolChainRecord = (chunk, traceDescription = null) => {
      const safeChunk = chunk || { role: MESSAGE_ROLES.ASSISTANT, tool_calls: [] }
      const traceRecord = createToolTraceRecord({
        index: null,
        chunk: safeChunk,
        traceDescription,
        pending: safeChunk.role === MESSAGE_ROLES.ASSISTANT
      })

      const record = createRecord({
        index: nextMessageIndex,
        message: { role: MESSAGE_ROLES.ASSISTANT, content: '', reasoning_content: '' },
        pending: true,
        renderMode: MESSAGE_RENDER_MODE.STREAM,
        expectReasoning: runOptions.thinkingModeEnabled,
        assistantRenderMode: ASSISTANT_DISPLAY_MODES.TOOL_CHAIN,
        enabledToolAPINames: runOptions.enabledToolAPINames,
        toolTraceRecords: [traceRecord]
      })

      if (nextMessageIndex !== null) {
        nextMessageIndex += 1
      }

      activeMessageKey = record.key
      activeRecordMode = ASSISTANT_DISPLAY_MODES.TOOL_CHAIN
      activeToolTraceKey = traceRecord.key
      pendingToolCallChunk = null
      createdRecordKeys.push(record.key)
      setMessages((prev) => [...prev, record])
    }

    const updateFinalAssistantInToolChain = (chunk) => {
      setMessages((prev) => prev.map((item) => {
        if (item.key !== activeMessageKey) {
          return item
        }

        const mergedMessage = mergeSchemaMessageChunk(item.message, chunk)
        const pending = !isMessageFinished(mergedMessage)
        if (!pending) {
          activeMessageKey = null
          activeRecordMode = ASSISTANT_DISPLAY_MODES.DEFAULT
          activeToolTraceKey = null
          pendingToolCallChunk = null
        }

        return {
          ...item,
          message: mergedMessage,
          pending
        }
      }))
    }

    const updateCurrentToolCallStatus = (chunk) => {
      if (!activeMessageKey || !activeToolTraceKey) {
        return
      }
      appendToolTraceToActiveRecord(chunk, { mergeIntoLast: true })
    }

    const updatePendingToolCallChunk = (chunk) => {
      pendingToolCallChunk = mergeSchemaMessageChunk(
        pendingToolCallChunk || { role: MESSAGE_ROLES.ASSISTANT, content: '', reasoning_content: '' },
        {
          ...chunk,
          content: '',
          reasoning_content: ''
        }
      )
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

          if (parsed.type === 'done') {
            finalizeStream()
            continue
          }

          if (parsed.type !== 'json') {
            continue
          }

          const payload = parsed.data
          if (payload?.type === SSE_EVENT_TYPES.META) {
            if (typeof payload.message_index === 'number') {
              nextMessageIndex = payload.message_index
            }
            continue
          }

          if (payload?.type === SSE_EVENT_TYPES.ERROR) {
            finalizeStream()
            message.error(payload.message || '流式传输出错')
            return
          }

          if (!isSchemaMessagePayload(payload)) {
            continue
          }

          const hasToolCalls = Array.isArray(payload.tool_calls) && payload.tool_calls.length > 0
          const hasVisibleContent = hasVisibleAssistantContent(payload)
          const finishReason = payload.response_meta?.finish_reason

          if (payload.role === MESSAGE_ROLES.ASSISTANT) {
            if (hasToolCalls) {
              updatePendingToolCallChunk(payload)
              continue
            }

            if (activeRecordMode === ASSISTANT_DISPLAY_MODES.TOOL_CHAIN) {
              if (finishReason === 'tool_calls') {
                if (pendingToolCallChunk) {
                  commitPendingToolCallToThoughtChain()
                } else {
                  updateCurrentToolCallStatus(payload)
                }
                continue
              }

              if (hasVisibleContent || finishReason === 'stop') {
                updateFinalAssistantInToolChain(payload)
                continue
              }

              if (isMessageFinished(payload)) {
                updateFinalAssistantInToolChain(payload)
              }
              continue
            }

            if (finishReason === 'tool_calls') {
              if (!activeMessageKey) {
                createToolChainRecord(pendingToolCallChunk || { role: MESSAGE_ROLES.ASSISTANT, tool_calls: [] })
              } else {
                commitPendingToolCallToThoughtChain()
              }
              continue
            }

            if (!activeMessageKey) {
              createAssistantRecord(payload)
              continue
            }

            updateActiveAssistantRecord(payload)
            continue
          }

          if (payload.role === MESSAGE_ROLES.TOOL) {
            if (!activeMessageKey) {
              createToolChainRecord(payload)
            } else {
              if (activeToolTraceKey) {
                setMessages((prev) => prev.map((item) => {
                  if (item.key !== activeMessageKey) {
                    return item
                  }

                  const toolTraceRecords = Array.isArray(item.toolTraceRecords) ? [...item.toolTraceRecords] : []
                  const traceIndex = toolTraceRecords.findIndex((record) => record.key === activeToolTraceKey)
                  if (traceIndex === -1) {
                    return item
                  }

                  toolTraceRecords[traceIndex] = {
                    ...toolTraceRecords[traceIndex],
                    pending: false
                  }

                  return {
                    ...item,
                    toolTraceRecords
                  }
                }))
              }
              appendToolTraceToActiveRecord(payload)
            }
          }
        }
      }

      finalizeStream()
    } catch (err) {
      console.error('Stream error:', err)
      finalizeStream()
      message.error('流式传输出错: ' + err.message)
    } finally {
      if (didCreateSession) {
        loadSessions()
      }
    }
  }, [activeKey, buildChatRequest, createServerSession, isTempSession, loadSessions, message])

  const sendGenerateMessage = useCallback(async (question, runOptions) => {
    const payload = buildChatRequest(question, runOptions)
    const hasEnabledTools = runOptions.enabledToolAPINames.length > 0
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
      const total = messages.length + 1
      const page = Math.ceil(total / MESSAGE_PAGE_SIZE)
      return prevPage !== page ? page : prevPage
    })

    const runOptions = buildChatRunOptions()

    if (isStreaming) {
      await sendStreamMessage(messageContent, runOptions)
    } else {
      await sendGenerateMessage(messageContent, runOptions)
    }

    bubbleListRef.current?.scrollTo?.({ top: 'bottom', behavior: 'smooth' })
  }, [attachments, buildChatRunOptions, handleAttachmentUpload, isStreaming, message, messages.length, sendGenerateMessage, sendStreamMessage])

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

  return {
    bubbleListRef,
    availableTools,
    enabledToolAPINames,
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
    setThinkingMode,
    setIsStreaming,
    setCurrentPage,
    setInputValue,
    setAttachments,
    setAttachmentsOpen
  }
}

export default useChat
