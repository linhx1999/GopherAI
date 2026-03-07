import { Button } from 'antd'
import { RollbackOutlined, SyncOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'

/**
 * 聊天页面顶部工具栏
 */
const ChatHeader = ({ canSyncHistory, onSyncHistory }) => {
  const navigate = useNavigate()

  return (
    <div className="chat-toolbar">
      <Button icon={<RollbackOutlined />} onClick={() => navigate('/menu')}>
        返回
      </Button>
      <Button icon={<SyncOutlined />} onClick={onSyncHistory} disabled={!canSyncHistory}>
        同步历史
      </Button>
    </div>
  )
}

export default ChatHeader
