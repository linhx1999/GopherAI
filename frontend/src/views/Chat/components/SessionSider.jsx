import { Button, Input } from 'antd'
import { PlusOutlined, EditOutlined, ShareAltOutlined, DeleteOutlined } from '@ant-design/icons'
import { Conversations } from '@ant-design/x'

/**
 * 左侧会话列表侧边栏
 */
const SessionSider = ({
  sessions,
  activeKey,
  editingSession,
  editTitle,
  onCreateSession,
  onSwitchSession,
  onMenuClick,
  onEditTitleChange,
  onConfirmRename
}) => {
  // 会话列表项
  const conversationItems = sessions.map(session => ({
    key: session.id,
    label: editingSession === session.id ? (
      <Input
        size="small"
        value={editTitle}
        onChange={(e) => onEditTitleChange(e.target.value)}
        onPressEnter={onConfirmRename}
        onBlur={onConfirmRename}
        autoFocus
        onClick={(e) => e.stopPropagation()}
      />
    ) : session.title || `会话 ${session.id}`,
    timestamp: session.timestamp
  }))

  return (
    <div className="session-sider">
      <div className="session-header">
        <span>会话列表</span>
        <Button type="primary" icon={<PlusOutlined />} onClick={onCreateSession} block>
          新聊天
        </Button>
      </div>
      <Conversations
        items={conversationItems}
        activeKey={activeKey}
        onActiveChange={onSwitchSession}
        className="conversations-list"
        menu={(item) => ({
          items: [
            { label: '重命名', key: 'rename', icon: <EditOutlined /> },
            { label: '分享', key: 'share', icon: <ShareAltOutlined /> },
            { type: 'divider' },
            { label: '删除会话', key: 'deleteChat', icon: <DeleteOutlined />, danger: true }
          ],
          onClick: (itemInfo) => onMenuClick(itemInfo, item.key)
        })}
      />
    </div>
  )
}

export default SessionSider
