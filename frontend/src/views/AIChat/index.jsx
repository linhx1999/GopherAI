import { useState, useRef, useCallback, useMemo, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Layout, Button, Select, Checkbox, App, Pagination, Input, Avatar, Typography, Badge } from 'antd'
import {
  RollbackOutlined,
  SyncOutlined,
  PlusOutlined,
  EditOutlined,
  ShareAltOutlined,
  DeleteOutlined,
  CopyOutlined,
  RedoOutlined,
  SoundOutlined,
  UserOutlined,
  RobotOutlined,
  CloudUploadOutlined,
  LinkOutlined
} from '@ant-design/icons'
import { Conversations, Sender, Welcome, Bubble, Actions, Attachments } from '@ant-design/x'
import XMarkdown from '@ant-design/x-markdown'
import api, { API_BASE_URL } from '../../utils/api'
import {
  MODEL_OPTIONS,
  MESSAGE_PAGE_SIZE,
  COLORS,
  MESSAGE_MAX_WIDTH,
  STATUS_CODES,
  API_ENDPOINTS,
  MESSAGE_ROLES,
  SPECIAL_SESSIONS
} from './config/constants'
import './index.css'

const { Text } = Typography
const { Sider, Content } = Layout

// ID 生成器
let idCounter = 0
const generateMessageId = () => {
  idCounter += 1
  return `msg_${Date.now()}_${idCounter}`
}

// 流式内容 Hook - 参考 Ant Design X 官方示例
// 用于实现逐字显示的打字效果
function useStreamContent(
  content,
  { step = 2, interval = 50 } = {}
) {
  const [streamContent, setStreamContent] = useState('')
  const streamRef = useRef('')
  const doneRef = useRef(true)
  const timerRef = useRef(-1)
  const stepRef = useRef(step)
  const intervalRef = useRef(interval)

  // 使用 useEffect 更新 ref
  useEffect(() => {
    stepRef.current = step
    intervalRef.current = interval
  }, [step, interval])

  // 流式开始函数
  const startStream = useCallback((text) => {
    doneRef.current = false
    streamRef.current = ''
    timerRef.current = setInterval(() => {
      const len = streamRef.current.length + stepRef.current
      if (len <= text.length - 1) {
        const newContent = text.slice(0, len)
        setStreamContent(newContent)
        streamRef.current = newContent
      } else {
        setStreamContent(text)
        streamRef.current = text
        doneRef.current = true
        clearInterval(timerRef.current)
      }
    }, intervalRef.current)
  }, [])

  useEffect(() => {
    // 内容相同，不处理
    if (content === streamRef.current) return

    // 清空内容
    if (!content && streamRef.current) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setStreamContent('')
      doneRef.current = true
      clearInterval(timerRef.current)
      return
    }

    // 新内容开始流式
    if (!streamRef.current && content) {
      clearInterval(timerRef.current)
      startStream(content)
    } else if (content.indexOf(streamRef.current) !== 0) {
      // 非起始子集认为是全新内容，重新开始流式
      clearInterval(timerRef.current)
      startStream(content)
    }
  }, [content, startStream])

  // 清理定时器
  useEffect(() => {
    return () => clearInterval(timerRef.current)
  }, [])

  // 使用 state 来跟踪 done 状态，而不是直接返回 ref
  const [isDone, setIsDone] = useState(true)
  useEffect(() => {
    // 定期检查 done 状态
    const checkDone = setInterval(() => {
      setIsDone(doneRef.current)
    }, 50)
    return () => clearInterval(checkDone)
  }, [])

  return [streamContent, isDone]
}

// 创建消息操作项
const createMessageActions = (isUser) => {
  const baseItems = [{ key: 'copy', icon: <CopyOutlined />, label: '复制' }]
  if (isUser) {
    return [...baseItems, { key: 'retry', icon: <RedoOutlined />, label: '重发' }]
  }
  return [...baseItems, { key: 'tts', icon: <SoundOutlined />, label: '朗读' }]
}

// Markdown 渲染
const renderMarkdown = (content) => <XMarkdown content={content} />

// 流式消息组件 - 使用 useStreamContent 实现逐字显示效果
const StreamBubble = ({ item, onActionClick }) => {
  const [streamContent, isDone] = useStreamContent(item.content, { step: 3, interval: 30 })
  const isStreaming = item.streaming && !isDone

  return (
    <Bubble
      placement="start"
      avatar={<Avatar icon={<RobotOutlined />} style={{ backgroundColor: COLORS.primary }} />}
      header={<Text type="secondary" style={{ fontSize: 12 }}>AI 助手</Text>}
      content={isStreaming ? streamContent : item.content}
      streaming={isStreaming}
      typing={false}
      contentRender={renderMarkdown}
      footer={item.loading ? null : (
        <Actions items={createMessageActions(false)} onClick={(info) => onActionClick(item, info)} />
      )}
      style={{ maxWidth: MESSAGE_MAX_WIDTH }}
    />
  )
}

// Role 配置
const createRoleConfig = (onActionClick) => ({
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

// 主组件
const AIChat = () => {
  const navigate = useNavigate()
  const { message } = App.useApp()
  const bubbleListRef = useRef(null)
  const senderRef = useRef(null)

  // 基础状态
  const [selectedModel, setSelectedModel] = useState('1')
  const [isStreaming, setIsStreaming] = useState(true)
  const [currentPage, setCurrentPage] = useState(1)
  const [inputValue, setInputValue] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [fileList, setFileList] = useState([])
  const [indexedFiles, setIndexedFiles] = useState([]) // 已索引的文件列表

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

  // 派生数据
  const paginatedMessages = useMemo(() => {
    const start = (currentPage - 1) * MESSAGE_PAGE_SIZE
    return messages.slice(start, start + MESSAGE_PAGE_SIZE)
  }, [messages, currentPage])

  // 动态生成模型选项（包括 RAG 文件选项）
  const modelOptions = useMemo(() => {
    const options = [...MODEL_OPTIONS]

    // 添加 RAG 文件选项
    indexedFiles.forEach(file => {
      options.push({
        label: `RAG - ${file.name}`,
        value: `2-${file.id}`
      })
    })

    return options
  }, [indexedFiles])

  const canSyncHistory = activeKey && !isTempSession

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

  // 加载文件列表
  const loadFileList = useCallback(async () => {
    try {
      const response = await api.get(API_ENDPOINTS.FILE_LIST)
      if (response.data?.status_code === STATUS_CODES.SUCCESS) {
        const files = (response.data.files || []).map(f => ({
          id: f.id,
          name: f.file_name,
          byte: f.file_size_bytes || 0,
          fileType: f.file_type,
          createdAt: f.created_at,
          indexStatus: f.index_status,
          indexMessage: f.index_message
        }))
        setFileList(files)

        // 过滤出已索引的文件
        const indexed = files.filter(f => f.indexStatus === 'indexed')
        setIndexedFiles(indexed)
      }
    } catch (error) {
      console.error('Load file list error:', error)
    }
  }, [])

  // 初始化加载
  useState(() => {
    loadSessions()
    loadFileList()
  }, [loadSessions, loadFileList])

  // 加载消息历史
  const loadMessages = useCallback(async (sessionId) => {
    try {
      const response = await api.post(API_ENDPOINTS.CHAT_HISTORY, { sessionId })
      if (response.data?.status_code === STATUS_CODES.SUCCESS && Array.isArray(response.data.history)) {
        const historyMessages = response.data.history.map(item => ({
          key: String(item.id || generateMessageId()),
          role: item.is_user ? MESSAGE_ROLES.USER : MESSAGE_ROLES.AI,
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

  // 同步历史
  const handleSyncHistory = useCallback(async () => {
    if (!activeKey || isTempSession) {
      message.warning('请选择已有会话进行同步')
      return
    }
    await loadMessages(activeKey)
  }, [activeKey, isTempSession, loadMessages, message])

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

  // 解析 SSE 数据行
  const parseSSELine = useCallback((line) => {
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
  }, [])

  // 流式发送消息
  const sendStreamMessage = useCallback(async (question, isNewSession) => {
    const aiMessageId = generateMessageId()
    setMessages(prev => [...prev, {
      key: aiMessageId,
      role: MESSAGE_ROLES.AI,
      content: '',
      loading: true,
      streaming: true
    }])

    // fetch 不像 axios 自动加 baseURL，需要手动添加 API_BASE_URL 前缀
    const endpoint = isNewSession || isTempSession
      ? API_ENDPOINTS.CHAT_STREAM_NEW_SESSION
      : API_ENDPOINTS.CHAT_STREAM
    const url = `${API_BASE_URL}${endpoint}`

    const body = isNewSession || isTempSession
      ? { question, modelType: selectedModel }
      : { question, modelType: selectedModel, sessionId: activeKey }

    let currentSessionId = activeKey
    let fullContent = ''
    let isFinalized = false

    // 更新消息内容的辅助函数
    const updateMessageContent = (messageId, content) => {
      setMessages(prev => prev.map(m =>
        m.key === messageId ? { ...m, content, loading: false, streaming: true } : m
      ))
    }

    // 完成消息的辅助函数
    const finalizeMessage = (messageId) => {
      if (isFinalized) return
      isFinalized = true
      console.log('[SSE] Finalizing message:', messageId)
      setMessages(prev => prev.map(m =>
        m.key === messageId ? { ...m, loading: false, streaming: false } : m
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
        if (done) {
          console.log('[SSE] Stream done')
          break
        }

        const chunk = decoder.decode(value, { stream: true })
        console.log('[SSE] Received chunk:', chunk.length, 'bytes')
        buffer += chunk
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          const parsed = parseSSELine(line)
          if (!parsed) continue

          if (parsed.type === 'done') {
            console.log('[SSE] Received DONE signal')
            finalizeMessage(aiMessageId)
          } else if (parsed.type === 'json') {
            if (parsed.data.sessionId) {
              currentSessionId = String(parsed.data.sessionId)
              console.log('[SSE] Received sessionId:', currentSessionId)
              handleSessionCreated(currentSessionId)
            }
          } else if (parsed.type === 'text') {
            console.log('[SSE] Received text chunk:', parsed.data)
            fullContent += parsed.data
            console.log('[SSE] Full content length:', fullContent.length)
            updateMessageContent(aiMessageId, fullContent)
          }
        }
      }

      // 处理 buffer 中剩余的数据
      if (buffer.trim()) {
        console.log('[SSE] Processing remaining buffer:', buffer)
        const parsed = parseSSELine(buffer)
        if (parsed) {
          if (parsed.type === 'done') {
            finalizeMessage(aiMessageId)
          } else if (parsed.type === 'json' && parsed.data.sessionId) {
            currentSessionId = String(parsed.data.sessionId)
            handleSessionCreated(currentSessionId)
          } else if (parsed.type === 'text') {
            fullContent += parsed.data
            updateMessageContent(aiMessageId, fullContent)
          }
        }
      }

      // 如果还没有 finalize（可能没有收到 [DONE]），现在 finalize
      if (!isFinalized) {
        console.log('[SSE] Stream ended without DONE, finalizing')
        finalizeMessage(aiMessageId)
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
  }, [selectedModel, activeKey, isTempSession, handleSessionCreated, parseSSELine, message])

  // 非流式发送消息
  const sendNormalMessage = useCallback(async (question, isNewSession) => {
    const aiMessageId = generateMessageId()
    setMessages(prev => [...prev, {
      key: aiMessageId,
      role: MESSAGE_ROLES.AI,
      content: '',
      loading: true
    }])

    const endpoint = isNewSession || isTempSession
      ? API_ENDPOINTS.CHAT_SEND_NEW_SESSION
      : API_ENDPOINTS.CHAT_SEND

    const payload = isNewSession || isTempSession
      ? { question, modelType: selectedModel }
      : { question, modelType: selectedModel, sessionId: activeKey }

    try {
      const response = await api.post(endpoint, payload)

      if (response.data?.status_code === STATUS_CODES.SUCCESS) {
        const aiContent = response.data.Information || response.data.information || ''
        setMessages(prev => prev.map(m =>
          m.key === aiMessageId ? { ...m, content: aiContent, loading: false } : m
        ))

        if (isNewSession || isTempSession) {
          const sessionId = String(response.data.sessionId)
          handleSessionCreated(sessionId)
        }
      } else {
        message.error(response.data?.status_msg || '发送失败')
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
  }, [selectedModel, activeKey, isTempSession, handleSessionCreated, message])

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
        loadFileList()
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

    const isNewSession = !activeKey
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
        await sendStreamMessage(messageContent, isNewSession)
      } else {
        await sendNormalMessage(messageContent, isNewSession)
      }
      bubbleListRef.current?.scrollTo?.({ top: 'bottom', behavior: 'smooth' })
    } catch (err) {
      console.error('Send message error:', err)
      message.error('发送失败，请重试')
    }
  }, [activeKey, messages.length, isStreaming, sendStreamMessage, sendNormalMessage, message, attachments, handleAttachmentUpload])

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
        // 轮询查询结果
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

  // 会话列表项
  const conversationItems = useMemo(() => {
    return sessions.map(session => ({
      key: session.id,
      label: editingSession === session.id ? (
        <Input
          size="small"
          value={editTitle}
          onChange={(e) => setEditTitle(e.target.value)}
          onPressEnter={confirmRename}
          onBlur={confirmRename}
          autoFocus
          onClick={(e) => e.stopPropagation()}
        />
      ) : session.title || `会话 ${session.id}`,
      timestamp: session.timestamp
    }))
  }, [sessions, editingSession, editTitle, confirmRename])

  // Role 配置
  const roleConfig = useMemo(() => createRoleConfig(handleActionClick), [handleActionClick])

  // 附件区域头部
  const senderHeader = useMemo(() => (
    <Sender.Header
      title="附件"
      open={attachmentsOpen}
      onOpenChange={setAttachmentsOpen}
      styles={{
        content: {
          padding: 0,
        },
      }}
    >
      <Attachments
        beforeUpload={() => false}
        items={attachments}
        onChange={({ file, fileList }) => {
          const updatedFileList = fileList.map(item => {
            if (item.uid === file.uid && file.status !== 'removed' && item.originFileObj) {
              // 清理旧 URL
              if (item.url?.startsWith('blob:')) {
                URL.revokeObjectURL(item.url)
              }
              // 创建预览 URL
              return {
                ...item,
                url: URL.createObjectURL(item.originFileObj)
              }
            }
            return item
          })
          setAttachments(updatedFileList)
        }}
        placeholder={type =>
          type === 'drop'
            ? {
                title: '拖放文件到此处',
              }
            : {
                icon: <CloudUploadOutlined />,
                title: '上传文件',
                description: '点击或拖拽文件到此区域上传',
              }
        }
        getDropContainer={() => senderRef.current?.nativeElement}
      />
    </Sender.Header>
  ), [attachmentsOpen, attachments])

  return (
    <Layout className="ai-chat-container">
      <Sider width={280} className="session-sider">
        <div className="session-header">
          <span>会话列表</span>
          <Button type="primary" icon={<PlusOutlined />} onClick={createSession} block>
            新聊天
          </Button>
        </div>
        <Conversations
          items={conversationItems}
          activeKey={activeKey}
          onActiveChange={switchSession}
          className="conversations-list"
          menu={(item) => ({
            items: [
              { label: '重命名', key: 'rename', icon: <EditOutlined /> },
              { label: '分享', key: 'share', icon: <ShareAltOutlined /> },
              { type: 'divider' },
              { label: '删除会话', key: 'deleteChat', icon: <DeleteOutlined />, danger: true }
            ],
            onClick: (itemInfo) => handleMenuClick(itemInfo, item.key)
          })}
        />
      </Sider>

      <Content className="chat-content">
        <div className="chat-toolbar">
          <Button icon={<RollbackOutlined />} onClick={() => navigate('/menu')}>
            返回
          </Button>
          <Button icon={<SyncOutlined />} onClick={handleSyncHistory} disabled={!canSyncHistory}>
            同步历史
          </Button>
          <span>选择模型：</span>
          <Select
            value={selectedModel}
            onChange={setSelectedModel}
            options={modelOptions}
            style={{ width: 200 }}
            placeholder="选择模型"
          />
          <Checkbox checked={isStreaming} onChange={(e) => setIsStreaming(e.target.checked)}>
            流式响应
          </Checkbox>
        </div>

        <div className="messages-container">
          {messages.length === 0 ? (
            <Welcome title="Hello! 我是你的智能助手，请问有什么我可以帮助你的吗？" />
          ) : (
            <div className="message-list-container">
              {paginatedMessages.map((item) => {
                // AI 消息使用 StreamBubble 组件
                if (item.role === MESSAGE_ROLES.AI) {
                  return (
                    <StreamBubble
                      key={item.key}
                      item={item}
                      onActionClick={handleActionClick}
                    />
                  )
                }
                // 其他消息使用标准 Bubble 组件
                const config = roleConfig[item.role] || roleConfig.user
                return (
                  <Bubble
                    key={item.key}
                    {...config}
                    content={item.content}
                    loading={item.loading}
                  />
                )
              })}
            </div>
          )}
        </div>

        {messages.length > MESSAGE_PAGE_SIZE && (
          <div className="pagination-container">
            <Pagination
              simple
              current={currentPage}
              onChange={setCurrentPage}
              total={messages.length}
              pageSize={MESSAGE_PAGE_SIZE}
              showSizeChanger={false}
            />
          </div>
        )}

        <div className="input-container">
          <Sender
            ref={senderRef}
            header={senderHeader}
            prefix={
              <Button
                onClick={() => setAttachmentsOpen(!attachmentsOpen)}
                icon={<LinkOutlined />}
              />
            }
            value={inputValue}
            onChange={setInputValue}
            onSubmit={handleSend}
            loading={isLoading}
            placeholder="请输入你的问题..."
          />
        </div>
      </Content>
    </Layout>
  )
}

export default AIChat
