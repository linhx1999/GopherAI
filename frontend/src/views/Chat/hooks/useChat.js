import { useState, useRef, useCallback, useEffect } from 'react'
import { App } from 'antd'
import api, { API_BASE_URL } from '../../../utils/api'
import {
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
  parseSSELine
} from '../utils/helpers.jsx'

const createRecord = ({ key = generateMessageId(), index = null, message, pending = false, createdAt = null }) => ({
  key,
  index,
  message,
  pending,
  createdAt
})

const useChat = () => {
  const { message } = App.useApp()
  const bubbleListRef = useRef(null)

  const [selectedTools, setSelectedTools] = useState([])
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
        const sessionItems = (response.data.data?.[0]?.sessions || [])
          .map((s) => ({
            id: String(s.sessionId),
            title: s.title || `会话 ${s.sessionId}`,
            createdAt: s.createdAt
          }))
          .sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt))
        setSessions(sessionItems)
      }
    } catch (error) {
      console.error('Load sessions error:', error)
    }
  }, [])

  useEffect(() => {
    loadSessions()
  }, [loadSessions])

  const loadMessages = useCallback(async (sessionId) => {
    try {
      const response = await api.get(API_ENDPOINTS.AGENT_MESSAGES(sessionId))
      if (response.data?.code === STATUS_CODES.SUCCESS && Array.isArray(response.data.data?.[0]?.messages)) {
        const historyMessages = response.data.data[0].messages.map((item) => createRecord({
          key: `msg_${item.index}_${item.message?.role || 'unknown'}`,
          index: item.index,
          message: item.message || {},
          pending: false,
          createdAt: item.created_at
        }))
        setMessages(historyMessages)
      } else {
        setMessages([])
      }
    } catch (err) {
      console.error('Load history error:', err)
      setMessages([])
    }
  }, [])

  const createSession = useCallback(() => {
    setSessions((prev) => prev.filter((s) => s.id !== SPECIAL_SESSIONS.TEMP))
    setSessions((prev) => [{ id: SPECIAL_SESSIONS.TEMP, title: '新会话', timestamp: Date.now() }, ...prev])
    setActiveKey(SPECIAL_SESSIONS.TEMP)
    setIsTempSession(true)
    setMessages([])
    setCurrentPage(1)
  }, [])

  const switchSession = useCallback(async (sessionId) => {
    if (!sessionId) return
    setActiveKey(sessionId)
    setIsTempSession(false)
    setCurrentPage(1)
    if (sessionId !== SPECIAL_SESSIONS.TEMP) {
      await loadMessages(sessionId)
    } else {
      setMessages([])
    }
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

  const handleSessionCreated = useCallback((sessionId) => {
    const newSid = String(sessionId)
    if (isTempSession) {
      setSessions((prev) => prev.map((s) => (
        s.id === SPECIAL_SESSIONS.TEMP ? { id: newSid, title: '新会话', timestamp: Date.now() } : s
      )))
      setActiveKey(newSid)
      setIsTempSession(false)
    } else if (!sessions.some((s) => s.id === newSid)) {
      loadSessions()
    }
  }, [isTempSession, sessions, loadSessions])

  const sendStreamMessage = useCallback(async (question, regenerateFrom = null) => {
    const url = `${API_BASE_URL}/${API_ENDPOINTS.AGENT}`
    const body = {
      message: question,
      tools: selectedTools,
      thinking_mode: thinkingMode,
      stream: true
    }

    if (activeKey && !isTempSession) {
      body.session_id = activeKey
    }
    if (regenerateFrom !== null) {
      body.regenerate_from = regenerateFrom
    }

    let nextMessageIndex = null
    let activeMessageKey = null
    const createdRecordKeys = []

    const finalizeStream = () => {
      if (createdRecordKeys.length > 0) {
        setMessages((prev) => prev.map((item) => (
          createdRecordKeys.includes(item.key) ? { ...item, pending: false } : item
        )))
      }
      activeMessageKey = null
      setIsLoading(false)
    }

    const appendStreamRecord = (chunk) => {
      const record = createRecord({
        index: nextMessageIndex,
        message: mergeSchemaMessageChunk({}, chunk),
        pending: !isMessageFinished(chunk)
      })
      if (nextMessageIndex !== null) {
        nextMessageIndex += 1
      }
      activeMessageKey = isMessageFinished(record.message) ? null : record.key
      createdRecordKeys.push(record.key)
      setMessages((prev) => [...prev, record])
    }

    const updateStreamRecord = (chunk) => {
      setMessages((prev) => prev.map((item) => {
        if (item.key !== activeMessageKey) {
          return item
        }
        const mergedMessage = mergeSchemaMessageChunk(item.message, chunk)
        const pending = !isMessageFinished(mergedMessage)
        if (!pending) {
          activeMessageKey = null
        }
        return {
          ...item,
          message: mergedMessage,
          pending
        }
      }))
    }

    try {
      const response = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token') || ''}`
        },
        body: JSON.stringify(body)
      })

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
            if (payload.session_id) {
              handleSessionCreated(String(payload.session_id))
            }
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

          if (!activeMessageKey) {
            appendStreamRecord(payload)
            continue
          }

          updateStreamRecord(payload)
        }
      }

      finalizeStream()
    } catch (err) {
      console.error('Stream error:', err)
      finalizeStream()
      message.error('流式传输出错: ' + err.message)
    }
  }, [selectedTools, thinkingMode, activeKey, isTempSession, handleSessionCreated, message])

  const sendNormalMessage = useCallback(async (question, regenerateFrom = null) => {
    const payload = {
      message: question,
      tools: selectedTools,
      thinking_mode: thinkingMode,
      stream: false
    }

    if (activeKey && !isTempSession) {
      payload.session_id = activeKey
    }
    if (regenerateFrom !== null) {
      payload.regenerate_from = regenerateFrom
    }

    try {
      const response = await api.post(API_ENDPOINTS.AGENT, payload)

      if (response.data?.code === STATUS_CODES.SUCCESS) {
        const data = response.data.data?.[0] || {}
        if (data.session_id) {
          handleSessionCreated(String(data.session_id))
          await loadMessages(String(data.session_id))
        } else if (activeKey && !isTempSession) {
          await loadMessages(activeKey)
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
  }, [selectedTools, thinkingMode, activeKey, isTempSession, handleSessionCreated, loadMessages, message])

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

  const handleSend = useCallback(async (content, options = {}) => {
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

    if (options.regenerateFrom === undefined) {
      const userRecord = createRecord({
        message: { role: MESSAGE_ROLES.USER, content: messageContent }
      })

      if (options.replaceIndex !== undefined) {
        setMessages((prev) => {
          const next = prev.slice(0, options.replaceIndex)
          next.push(userRecord)
          return next
        })
      } else {
        setMessages((prev) => [...prev, userRecord])
      }
    }

    setInputValue('')
    attachments.forEach((item) => {
      if (item.url?.startsWith('blob:')) {
        URL.revokeObjectURL(item.url)
      }
    })
    setAttachments([])
    setAttachmentsOpen(false)

    setCurrentPage((prevPage) => {
      const total = options.replaceIndex !== undefined ? options.replaceIndex + 1 : messages.length + 1
      const page = Math.ceil(total / MESSAGE_PAGE_SIZE)
      return prevPage !== page ? page : prevPage
    })

    if (isStreaming) {
      await sendStreamMessage(messageContent, options.regenerateFrom)
    } else {
      await sendNormalMessage(messageContent, options.regenerateFrom)
    }

    bubbleListRef.current?.scrollTo?.({ top: 'bottom', behavior: 'smooth' })
  }, [attachments, handleAttachmentUpload, isStreaming, message, messages.length, sendNormalMessage, sendStreamMessage])

  const handleRetry = useCallback((record) => {
    if (record.message?.role !== MESSAGE_ROLES.USER) return
    const index = messages.findIndex((item) => item.key === record.key)
    if (index === -1) return
    handleSend(record.message.content || '', { replaceIndex: index })
    setCurrentPage(Math.ceil(messages.length / MESSAGE_PAGE_SIZE))
  }, [messages, handleSend])

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
      case 'retry':
        handleRetry(record)
        break
      default:
        break
    }
  }, [playTTS, handleRetry])

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
    selectedTools,
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
    setSelectedTools,
    setThinkingMode,
    setIsStreaming,
    setCurrentPage,
    setInputValue,
    setAttachments,
    setAttachmentsOpen
  }
}

export default useChat
