import { useState, useRef, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { Layout, Button, Upload, App } from 'antd'
import { RollbackOutlined, UploadOutlined } from '@ant-design/icons'
import { Bubble } from '@ant-design/x'
import api from '../utils/api'
import './ImageRecognition.css'

const { Sider, Content } = Layout

const ImageRecognition = () => {
  const navigate = useNavigate()
  const { message } = App.useApp()
  const [messages, setMessages] = useState([])
  const [selectedFile, setSelectedFile] = useState(null)
  const [previewUrl, setPreviewUrl] = useState(null)
  const [uploading, setUploading] = useState(false)
  const messagesEndRef = useRef(null)

  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [])

  const handleFileChange = (info) => {
    const file = info.file
    if (file) {
      setSelectedFile(file)
      setPreviewUrl(URL.createObjectURL(file))
    }
  }

  const handleSubmit = async () => {
    if (!selectedFile) {
      message.warning('请先选择图片')
      return
    }

    const imageUrl = previewUrl

    // 添加用户消息到UI
    const userMessage = {
      id: Date.now().toString(),
      role: 'user',
      content: `已上传图片: ${selectedFile.name}`,
      imageUrl: imageUrl
    }

    setMessages(prev => [...prev, userMessage])
    setSelectedFile(null)
    setPreviewUrl(null)

    // 创建FormData
    const formData = new FormData()
    formData.append('image', selectedFile)

    try {
      setUploading(true)
      const response = await api.post('/image/recognize', formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        }
      })

      if (response.data?.class_name) {
        const aiMessage = {
          id: Date.now().toString() + '_ai',
          role: 'assistant',
          content: `识别结果: ${response.data.class_name}`
        }
        setMessages(prev => [...prev, aiMessage])
      } else {
        const errorMessage = {
          id: Date.now().toString() + '_ai',
          role: 'assistant',
          content: `[错误] ${response.data?.status_msg || '识别失败'}`
        }
        setMessages(prev => [...prev, errorMessage])
      }
    } catch (error) {
      console.error('Upload error:', error)
      const errorMessage = {
        id: Date.now().toString() + '_ai',
        role: 'assistant',
        content: `[错误] 无法连接到服务器或上传失败: ${error.message}`
      }
      setMessages(prev => [...prev, errorMessage])
    } finally {
      URL.revokeObjectURL(imageUrl)
      setUploading(false)
      setTimeout(scrollToBottom, 100)
    }
  }

  // 渲染消息气泡
  const renderMessage = (msg) => {
    const isUser = msg.role === 'user'
    return (
      <Bubble
        key={msg.id}
        placement={isUser ? 'end' : 'start'}
        content={
          <div>
            <div>{msg.content}</div>
            {msg.imageUrl && (
              <img
                src={msg.imageUrl}
                alt="上传的图片"
                style={{
                  maxWidth: '250px',
                  borderRadius: '12px',
                  marginTop: '12px',
                  boxShadow: '0 4px 15px rgba(0, 0, 0, 0.2)'
                }}
              />
            )}
          </div>
        }
        avatar={isUser ? undefined : <span style={{ fontSize: 24 }}>🤖</span>}
      />
    )
  }

  return (
    <Layout className="image-recognition-container">
      {/* 左侧会话列表 */}
      <Sider width={280} className="session-sider">
        <div className="session-header">
          <span>图像识别</span>
        </div>
        <div className="session-list">
          <div className="session-item active">
            图像识别助手
          </div>
        </div>
      </Sider>

      {/* 右侧聊天区域 */}
      <Content className="chat-content">
        {/* 顶部工具栏 */}
        <div className="chat-toolbar">
          <Button
            icon={<RollbackOutlined />}
            onClick={() => navigate('/menu')}
          >
            返回
          </Button>
          <h2>AI 图像识别助手</h2>
        </div>

        {/* 消息列表 */}
        <div className="messages-container">
          {messages.map(renderMessage)}
          <div ref={messagesEndRef} />
        </div>

        {/* 输入区域 */}
        <div className="input-container">
          <div className="upload-section">
            <Upload
              accept="image/*"
              beforeUpload={() => false}
              onChange={handleFileChange}
              showUploadList={false}
              maxCount={1}
            >
              <Button icon={<UploadOutlined />}>
                选择图片
              </Button>
            </Upload>
            {previewUrl && (
              <div className="preview-section">
                <img
                  src={previewUrl}
                  alt="预览"
                  className="preview-image"
                />
              </div>
            )}
            <Button
              type="primary"
              onClick={handleSubmit}
              loading={uploading}
              disabled={!selectedFile}
            >
              发送图片
            </Button>
          </div>
        </div>
      </Content>
    </Layout>
  )
}

export default ImageRecognition
