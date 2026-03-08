import { Avatar, Typography } from 'antd'
import { useEffect } from 'react'
import { RobotOutlined } from '@ant-design/icons'
import { Bubble, Actions, Think } from '@ant-design/x'
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
const StreamBubble = ({ item, onActionClick, onReasoningDisplayComplete }) => {
  const [streamContent, isDone] = useStreamContent(item.content, { step: 3, interval: 30 })
  const [streamReasoning, isReasoningDone] = useStreamContent(item.reasoningContent || '', { step: 3, interval: 30 })
  const isStreaming = !isDone
  const isReasoningStreaming = item.streaming && !item.reasoningCompleted
  const showReasoning = Boolean(item.reasoningContent)
  const showAnswer = item.answerUnlocked && (Boolean(item.content) || !showReasoning)
  const reasoningDisplayContent = isReasoningDone ? item.reasoningContent : streamReasoning
  const answerDisplayContent = isDone ? item.content : streamContent

  useEffect(() => {
    if (showReasoning && item.reasoningCompleted && isReasoningDone && !item.answerUnlocked) {
      onReasoningDisplayComplete?.(item.key)
    }
  }, [showReasoning, item.reasoningCompleted, isReasoningDone, item.answerUnlocked, item.key, onReasoningDisplayComplete])

  return (
    <div className="assistant-message">
      {showReasoning && (
        <Think
          title="深度思考"
          loading={isReasoningStreaming}
          expanded={item.streaming ? true : undefined}
          defaultExpanded
          className="assistant-thinking"
          styles={{
            content: {
              background: 'rgba(255, 255, 255, 0.72)',
              border: '1px solid rgba(22, 119, 255, 0.08)',
              borderRadius: 14,
            }
          }}
        >
          {renderMarkdown(reasoningDisplayContent)}
        </Think>
      )}

      {showAnswer && (
        <Bubble
          placement="start"
          avatar={<Avatar icon={<RobotOutlined />} style={{ backgroundColor: COLORS.primary }} />}
          header={<Text type="secondary" style={{ fontSize: 12 }}>AI 助手</Text>}
          content={answerDisplayContent}
          streaming={isStreaming}
          typing={false}
          contentRender={renderMarkdown}
          footer={item.loading ? null : (
            <Actions items={createMessageActions(false)} onClick={(info) => onActionClick(item, info)} />
          )}
          style={{ maxWidth: MESSAGE_MAX_WIDTH }}
        />
      )}
    </div>
  )
}

export default StreamBubble
