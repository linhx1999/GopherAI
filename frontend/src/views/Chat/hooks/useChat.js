import { useState, useRef, useCallback, useMemo, useEffect } from 'react'
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
import { generateMessageId, parseSSELine } from '../utils/helpers.jsx'

/**
 * 聊天逻辑 Hook - 合并会话管理和消息发送逻辑
 */
const useChat = () => {
  const { message } = App.useApp()
  const bubbleListRef = useRef(null)

  // 基础状态
  const [selectedTools, setSelectedTools] = useState(['knowledge_search'])
  const [isStreaming, setIsStreaming] = useState(true)
  const [currentPage, setCurrentPage] = useState(1)
  const [inputValue, setInputValue] = useState('')
  const [isLoading, setIsLoading] = useState(false)

  // 附件状态
  const [attachments, setAttachments] = useState([])
  const [attachmentsOpen, setAttachmentsOpen] = useState(false)

  // 消息状态
  const [messages, setMessages] = useState([])

  // 会话状态
  const [sessions, setSessions] = useState([])
  const [activeKey, setActiveKey] = useState(null)
  const [isTempSession, setIsTempSession] = useState(false)
  const [editingSession, setEditingSession] = useState(null)
  const [editTitle, setEditTitle] = useState('')

  // ==================== 会话管理 ====================

  // 加载会话列表
  const loadSessions = useCallback(async () => {
    try {
      const response = await api.get(API_ENDPOINTS.SESSIONS)
      if (response.data?.status_code === STATUS_CODES.SUCCESS) {
        const sessionItems = (response.data.sessions || [])
          .map(s => ({
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

  // 初始化加载
  useEffect(() => {
    loadSessions()
  }, [loadSessions])

  // 加载消息历史
  const loadMessages = useCallback(async (sessionId) => {
    try {
      const response = await api.get(API_ENDPOINTS.AGENT_MESSAGES(sessionId))
      if (response.data?.code === STATUS_CODES.SUCCESS && Array.isArray(response.data.data?.messages)) {
        const historyMessages = response.data.data.messages.map(item => ({
          key: String(item.index || generateMessageId()),
          role: item.role === 'user' ? MESSAGE_ROLES.USER : MESSAGE_ROLES.AI,
          content: item.content
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

  // 创建新会话
  const createSession = useCallback(() => {
    setSessions(prev => prev.filter(s => s.id !== SPECIAL_SESSIONS.TEMP))
    setSessions(prev => [{ id: SPECIAL_SESSIONS.TEMP, title: '新会话', timestamp: Date.now() }, ...prev])
    setActiveKey(SPECIAL_SESSIONS.TEMP)
    setIsTempSession(true)
    setMessages([])
    setCurrentPage(1)
  }, [])

  // 切换会话
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

  // 删除会话
  const deleteSession = useCallback(async (sessionId) => {
    try {
      const response = await api.delete(API_ENDPOINTS.SESSION_DELETE(sessionId))
      if (response.data?.status_code === STATUS_CODES.SUCCESS) {
        message.success('会话已删除')
        setSessions(prev => prev.filter(s => s.id !== sessionId))
        if (activeKey === sessionId) {
          setActiveKey(null)
          setIsTempSession(false)
          setMessages([])
        }
      } else {
        message.error(response.data?.status_msg || '删除失败')
      }
    } catch (error) {
      console.error('Delete session error:', error)
      message.error('删除会话失败')
    }
  }, [activeKey, message])

  // 更新会话标题
  const updateSessionTitle = useCallback(async (sessionId, title) => {
    try {
      const response = await api.put(API_ENDPOINTS.SESSION_TITLE(sessionId), { title })
      if (response.data?.status_code === STATUS_CODES.SUCCESS) {
        message.success('标题已更新')
        setSessions(prev => prev.map(s => s.id === sessionId ? { ...s, title } : s))
      } else {
        message.error(response.data?.status_msg || '更新失败')
      }
    } catch (error) {
      console.error('Update title error:', error)
      message.error('更新标题失败')
    }
  }, [message])

  // 分享会话
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

  // 确认重命名
  const confirmRename = useCallback(async () => {
    if (!editingSession || !editTitle.trim()) {
      setEditingSession(null)
      return
    }
    await updateSessionTitle(editingSession, editTitle.trim())
    setEditingSession(null)
    setEditTitle('')
  }, [editingSession, editTitle, updateSessionTitle])

  // 菜单点击处理
  const handleMenuClick = useCallback((itemInfo, sessionId) => {
    itemInfo.domEvent.stopPropagation()
    const { key } = itemInfo

    switch (key) {
      case 'rename': {
        const session = sessions.find(s => s.id === sessionId)
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

  // 处理会话创建回调
  const handleSessionCreated = useCallback((sessionId) => {
    const newSid = String(sessionId)
    if (isTempSession) {
      setSessions(prev => prev.map(s =>
        s.id === SPECIAL_SESSIONS.TEMP ? { id: newSid, title: '新会话', timestamp: Date.now() } : s
      ))
      setActiveKey(newSid)
      setIsTempSession(false)
    }
  }, [isTempSession])

  // ==================== 消息发送 ====================

  // 流式发送消息
  const sendStreamMessage = useCallback(async (question, regenerateFrom = null) => {
    const aiMessageId = generateMessageId()
    setMessages(prev => [...prev, {
      key: aiMessageId,
      role: MESSAGE_ROLES.AI,
      content: '',
      loading: true,
      streaming: true
    }])

    const url = `${API_BASE_URL}/${API_ENDPOINTS.AGENT}`
    const body = {
      message: question,
      tools: selectedTools,
      stream: true
    }
    
    if (activeKey && !isTempSession) {
      body.session_id = activeKey
    }
    
    if (regenerateFrom !== null) {
      body.regenerate_from = regenerateFrom
    }

    let currentSessionId = activeKey
    let fullContent = ''
    let isFinalized = false

    const updateMessageContent = (content) => {
      setMessages(prev => prev.map(m =>
        m.key === aiMessageId ? { ...m, content, loading: false, streaming: true } : m
      ))
    }

    const finalizeMessage = () => {
      if (isFinalized) return
      isFinalized = true
      setMessages(prev => prev.map(m =>
        m.key === aiMessageId ? { ...m, loading: false, streaming: false } : m
      ))
      setIsLoading(false)
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

        const chunk = decoder.decode(value, { stream: true })
        buffer += chunk
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          const parsed = parseSSELine(line)
          if (!parsed) continue

          if (parsed.type === 'json' && parsed.data.type) {
            const event = parsed.data
            
            switch (event.type) {
              case SSE_EVENT_TYPES.META:
                if (event.session_id) {
                  currentSessionId = String(event.session_id)
                  handleSessionCreated(currentSessionId)
                }
                break
                
              case SSE_EVENT_TYPES.TOOL_CALL:
                console.log('[SSE] Tool call:', event)
                break
                
              case SSE_EVENT_TYPES.CONTENT_DELTA:
                if (event.content) {
                  fullContent += event.content
                  updateMessageContent(fullContent)
                }
                break
                
              case SSE_EVENT_TYPES.MESSAGE_END:
                finalizeMessage()
                break
                
              case SSE_EVENT_TYPES.ERROR:
                setMessages(prev => prev.map(m =>
                  m.key === aiMessageId ? { 
                    ...m, 
                    loading: false, 
                    streaming: false, 
                    content: fullContent + '\n\n[错误: ' + (event.message || '未知错误') + ']' 
                  } : m
                ))
                setIsLoading(false)
                return
                
              default:
                break
            }
          } else if (parsed.type === 'done') {
            finalizeMessage()
          } else if (parsed.type === 'json' && parsed.data.sessionId) {
            currentSessionId = String(parsed.data.sessionId)
            handleSessionCreated(currentSessionId)
          } else if (parsed.type === 'text') {
            fullContent += parsed.data
            updateMessageContent(fullContent)
          }
        }
      }

      if (!isFinalized) {
        finalizeMessage()
      }
    } catch (err) {
      console.error('Stream error:', err)
      if (!isFinalized) {
        setMessages(prev => prev.map(m =>
          m.key === aiMessageId ? { ...m, loading: false, streaming: false, content: '流式传输出错: ' + err.message } : m
        ))
        setIsLoading(false)
      }
      message.error('流式传输出错: ' + err.message)
    }
  }, [selectedTools, activeKey, isTempSession, handleSessionCreated, message])

  // 非流式发送消息
  const sendNormalMessage = useCallback(async (question, regenerateFrom = null) => {
    const aiMessageId = generateMessageId()
    setMessages(prev => [...prev, {
      key: aiMessageId,
      role: MESSAGE_ROLES.AI,
      content: '',
      loading: true
    }])

    const payload = {
      message: question,
      tools: selectedTools,
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
        const data = response.data.data || response.data
        const aiContent = data.content || ''
        setMessages(prev => prev.map(m =>
          m.key === aiMessageId ? { ...m, content: aiContent, loading: false } : m
        ))

        if (data.session_id) {
          handleSessionCreated(data.session_id)
        }
      } else {
        message.error(response.data?.msg || '发送失败')
        setMessages(prev => prev.map(m =>
          m.key === aiMessageId ? { ...m, loading: false, content: '发送失败' } : m
        ))
      }
    } catch (err) {
      console.error('Send message error:', err)
      message.error('发送失败，请重试')
      setMessages(prev => prev.map(m =>
        m.key === aiMessageId ? { ...m, loading: false, content: '发送失败' } : m
      ))
    }

    setIsLoading(false)
  }, [selectedTools, activeKey, isTempSession, handleSessionCreated, message])

  // 附件上传处理
  const handleAttachmentUpload = useCallback(async (file) => {
    const formData = new FormData()
    formData.append('file', file)
    try {
      const response = await api.post(API_ENDPOINTS.FILE_UPLOAD, formData, {
        headers: { 'Content-Type': 'multipart/form-data' }
      })
      if (response.data?.status_code === STATUS_CODES.SUCCESS) {
        message.success(`${file.name} 上传成功`)
        return response.data.file_id
      } else {
        message.error(response.data?.status_msg || '上传失败')
        return null
      }
    } catch (error) {
      console.error('Attachment upload error:', error)
      message.error(`${file.name} 上传失败`)
      return null
    }
  }, [message])

  // 处理发送
  const handleSend = useCallback(async (content, options = {}) => {
    if (!content.trim() && attachments.length === 0) {
      message.warning('请输入消息内容或添加附件')
      return
    }

    setIsLoading(true)

    // 上传附件
    let uploadedFileIds = []
    if (attachments.length > 0) {
      const uploadPromises = attachments
        .filter(item => item.originFileObj)
        .map(item => handleAttachmentUpload(item.originFileObj))
      const results = await Promise.all(uploadPromises)
      uploadedFileIds = results.filter(id => id !== null)
    }

    // 构建消息内容
    let messageContent = content
    if (uploadedFileIds.length > 0) {
      const fileNames = attachments.map(item => item.name).join(', ')
      messageContent = content ? `${content}\n\n[附件: ${fileNames}]` : `[附件: ${fileNames}]`
    }

    // 添加用户消息
    if (options.regenerateFrom === undefined) {
      const userMessage = {
        key: generateMessageId(),
        role: MESSAGE_ROLES.USER,
        content: messageContent
      }

      if (options.replaceIndex !== undefined) {
        setMessages(prev => {
          const newMessages = prev.slice(0, options.replaceIndex)
          newMessages.push(userMessage)
          return newMessages
        })
      } else {
        setMessages(prev => [...prev, userMessage])
      }
    }

    setInputValue('')
    // 清理附件
    attachments.forEach(item => {
      if (item.url?.startsWith('blob:')) {
        URL.revokeObjectURL(item.url)
      }
    })
    setAttachments([])
    setAttachmentsOpen(false)

    // 跳转到最后一页
    setCurrentPage(prevPage => {
      const newTotalPages = options.replaceIndex !== undefined
        ? Math.ceil((options.replaceIndex + 1) / MESSAGE_PAGE_SIZE)
        : Math.ceil((messages.length + 1) / MESSAGE_PAGE_SIZE)
      return prevPage !== newTotalPages ? newTotalPages : prevPage
    })

    try {
      if (isStreaming) {
        await sendStreamMessage(messageContent, options.regenerateFrom)
      } else {
        await sendNormalMessage(messageContent, options.regenerateFrom)
      }
      bubbleListRef.current?.scrollTo?.({ top: 'bottom', behavior: 'smooth' })
    } catch (err) {
      console.error('Send message error:', err)
      message.error('发送失败，请重试')
    }
  }, [messages.length, isStreaming, sendStreamMessage, sendNormalMessage, message, attachments, handleAttachmentUpload])

  // 重试
  const handleRetry = useCallback((msg) => {
    if (msg.role !== MESSAGE_ROLES.USER) return
    const index = messages.findIndex(m => m.key === msg.key)
    if (index === -1) return
    handleSend(msg.content, { replaceIndex: index })
    setCurrentPage(Math.ceil(messages.length / MESSAGE_PAGE_SIZE))
  }, [messages, handleSend])

  // TTS 播放
  const playTTS = useCallback(async (text) => {
    try {
      const response = await api.post(API_ENDPOINTS.TTS, { text })
      if (response.data?.status_code === STATUS_CODES.SUCCESS && response.data.task_id) {
        const taskId = response.data.task_id
        let attempts = 0
        while (attempts < 30) {
          const queryResponse = await api.get(API_ENDPOINTS.TTS_QUERY, { params: { task_id: taskId } })
          if (queryResponse.data?.status_code === STATUS_CODES.SUCCESS) {
            if (queryResponse.data.task_status === 'Success' && queryResponse.data.task_result) {
              const audio = new Audio(queryResponse.data.task_result)
              audio.play()
              return
            }
            if (queryResponse.data.task_status === 'Failed') {
              message.error('语音合成失败')
              return
            }
          }
          attempts++
          await new Promise(resolve => setTimeout(resolve, 2000))
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

  // 操作按钮点击
  const handleActionClick = useCallback((item, { key }) => {
    switch (key) {
      case 'copy':
        navigator.clipboard.writeText(item.content)
        break
      case 'tts':
        playTTS(item.content)
        break
      case 'retry':
        handleRetry(item)
        break
    }
  }, [playTTS, handleRetry])

  // 清理 blob URL
  useEffect(() => {
    return () => {
      attachments.forEach(item => {
        if (item.url?.startsWith('blob:')) {
          URL.revokeObjectURL(item.url)
        }
      })
    }
  }, [attachments])

  return {
    // refs
    bubbleListRef,
    // 基础状态
    selectedTools,
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
    // 会话操作
    createSession,
    switchSession,
    handleMenuClick,
    confirmRename,
    setEditTitle,
    // 消息操作
    handleSend,
    handleActionClick,
    // 状态更新
    setSelectedTools,
    setIsStreaming,
    setCurrentPage,
    setInputValue,
    setAttachments,
    setAttachmentsOpen
  }
}

export default useChat
