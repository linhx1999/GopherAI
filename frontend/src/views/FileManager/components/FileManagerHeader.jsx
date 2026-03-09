import { App, Button, Modal, Space } from 'antd'
import { LogoutOutlined, RollbackOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'

const FileManagerHeader = () => {
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
    <div className="file-manager-toolbar">
      <Space wrap>
        <Button icon={<RollbackOutlined />} onClick={() => navigate('/menu')}>
          返回
        </Button>
        <Button danger icon={<LogoutOutlined />} onClick={handleLogout}>
          退出登录
        </Button>
      </Space>

      <div className="file-manager-toolbar-copy">
        <span className="file-manager-toolbar-eyebrow">Knowledge Base</span>
        <h1>文件管理</h1>
        <p>统一管理知识库文件、索引状态和删除操作。</p>
      </div>
    </div>
  )
}

export default FileManagerHeader
