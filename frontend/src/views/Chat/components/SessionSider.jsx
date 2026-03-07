import { Input } from 'antd'
import { EditOutlined, ShareAltOutlined, DeleteOutlined } from '@ant-design/icons'
import { Conversations } from '@ant-design/x'
import { theme } from 'antd';

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
  const { token } = theme.useToken();
  const style = {
    height: '100vh',
    background: token.colorBgContainer,
    borderRadius: token.borderRadius,
  };
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
      <Conversations
        creation={{
          onClick: onCreateSession,
        }}
        style={style}
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
  )
}

export default SessionSider
