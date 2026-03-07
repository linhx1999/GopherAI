import { Avatar, Typography } from 'antd'
import { RobotOutlined } from '@ant-design/icons'
import { Bubble, Actions } from '@ant-design/x'
import XMarkdown from '@ant-design/x-markdown'
import useStreamContent from '../hooks/useStreamContent'
import { COLORS, MESSAGE_MAX_WIDTH } from '../config/constants'
import { createMessageActions } from '../utils/helpers.jsx'

const { Text } = Typography

// Markdown 渲染
const renderMarkdown = (content) => <XMarkdown content={content} />

/**
 * 流式消息组件 - 使用 useStreamContent 实现逐字显示效果
 */
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

export default StreamBubble
