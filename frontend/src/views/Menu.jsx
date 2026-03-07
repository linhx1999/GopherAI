import { useNavigate } from 'react-router-dom'
import { Layout, Card, Button, Modal, App } from 'antd'
import { MessageOutlined, CameraOutlined, FolderOutlined, LogoutOutlined } from '@ant-design/icons'
import './Menu.css'

const { Header, Content } = Layout

const Menu = () => {
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

  const menuItems = [
    {
      key: 'ai-chat',
      title: 'AI聊天',
      description: '与AI进行智能对话',
      icon: <MessageOutlined style={{ fontSize: 48, color: '#1677ff' }} />,
      path: '/ai-chat'
    },
    {
      key: 'image-recognition',
      title: '图像识别',
      description: '上传图片进行AI识别',
      icon: <CameraOutlined style={{ fontSize: 48, color: '#52c41a' }} />,
      path: '/image-recognition'
    },
    {
      key: 'file-manager',
      title: '文件管理',
      description: '管理知识库文件和索引',
      icon: <FolderOutlined style={{ fontSize: 48, color: '#faad14' }} />,
      path: '/file-manager'
    }
  ]

  return (
    <Layout className="menu-container">
      <Header className="menu-header">
        <h1>AI应用平台</h1>
        <Button
          type="primary"
          danger
          icon={<LogoutOutlined />}
          onClick={handleLogout}
        >
          退出登录
        </Button>
      </Header>
      <Content className="menu-content">
        <div className="menu-grid">
          {menuItems.map((item, index) => (
            <Card
              key={item.key}
              className="menu-item"
              bordered={false}
              onClick={() => navigate(item.path)}
              style={{ animationDelay: `${index * 0.1}s` }}
            >
              <div className="card-content">
                {item.icon}
                <h3>{item.title}</h3>
                <p>{item.description}</p>
              </div>
            </Card>
          ))}
        </div>
      </Content>
    </Layout>
  )
}

export default Menu
