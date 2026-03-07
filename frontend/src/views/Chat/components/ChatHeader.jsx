import { Button, Modal, App } from 'antd'
import { RollbackOutlined, LogoutOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'

/**
 * 聊天页面顶部工具栏
 */
const ChatHeader = () => {
  const navigate = useNavigate()
  const { message } = App.useApp()

  const handleLogout = () => {
    Modal.confirm({
      title: '提示',
      content: '确定要退出登录吗？',
      okText: '确定',
      cancelText: '取消',
      onOk: () => {
        localStorage.removeItem('token')
        message.success('退出登录成功')
        navigate('/login')
      }
    })
  }

  return (
    <div className="chat-toolbar">
      <Button icon={<RollbackOutlined />} onClick={() => navigate('/menu')}>
        返回
      </Button>
      <Button danger icon={<LogoutOutlined />} onClick={handleLogout}>
        退出登录
      </Button>
    </div>
  )
}

export default ChatHeader
