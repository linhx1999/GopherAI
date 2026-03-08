import { Avatar, Typography } from 'antd'
import { RobotOutlined } from '@ant-design/icons'
import { Actions, Bubble, Think, ThoughtChain } from '@ant-design/x'
import useStreamContent from '../hooks/useStreamContent'
import { COLORS, MESSAGE_MAX_WIDTH } from '../config/constants'
import {
  buildThoughtChainItems,
  createMessageActions,
  renderMarkdown,
  shouldUseToolChain
} from '../utils/helpers.jsx'

const { Text } = Typography

const ToolThoughtChain = ({ record = null, processes = [] }) => {
  const items = buildThoughtChainItems({ record, processRecords: processes })
  if (!items.length) {
    return null
  }

  return (
    <ThoughtChain
      className="assistant-thought-chain"
      items={items}
      line="dashed"
    />
  )
}

const AssistantBubble = ({ record, processes, onActionClick }) => {
  const message = record.message || {}
  const shouldAnimate = record.renderMode === 'stream' && record.pending
  const useToolChain = shouldUseToolChain(record, processes)
  const rawReasoningContent = message.reasoning_content || ''
  const rawAnswerContent = message.content || ''
  const [streamContent, isContentDone] = useStreamContent(rawAnswerContent, {
    step: 3,
    interval: 30,
    enabled: shouldAnimate
  })
  const [streamReasoning, isReasoningDone] = useStreamContent(rawReasoningContent, {
    step: 3,
    interval: 30,
    enabled: shouldAnimate
  })
  const reasoningDisplayContent = shouldAnimate && !isReasoningDone ? streamReasoning : rawReasoningContent
  const answerDisplayContent = shouldAnimate && !isContentDone ? streamContent : rawAnswerContent
  const isStreaming = shouldAnimate && !isContentDone
  const showReasoningPlaceholder = Boolean(record.expectReasoning) && shouldAnimate && !rawReasoningContent.trim() && !rawAnswerContent.trim()
  const showReasoning = !useToolChain && (Boolean(rawReasoningContent) || showReasoningPlaceholder)
  const showAnswer = Boolean(rawAnswerContent)
  const reasoningLoading = showReasoning && shouldAnimate && !rawAnswerContent.trim()

  return (
    <div className="assistant-message">
      {useToolChain ? <ToolThoughtChain record={record} processes={processes} /> : null}

      {showReasoning ? (
        <Think
          title="深度思考"
          loading={reasoningLoading}
          expanded
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
      ) : null}

      {showAnswer ? (
        <Bubble
          placement="start"
          avatar={<Avatar icon={<RobotOutlined />} style={{ backgroundColor: COLORS.primary }} />}
          header={<Text type="secondary" style={{ fontSize: 12 }}>AI 助手</Text>}
          content={answerDisplayContent}
          streaming={isStreaming}
          typing={false}
          contentRender={renderMarkdown}
          footer={record.pending ? null : (
            <Actions items={createMessageActions(false)} onClick={(info) => onActionClick(record, info)} />
          )}
          style={{ maxWidth: MESSAGE_MAX_WIDTH }}
        />
      ) : null}
    </div>
  )
}

export default AssistantBubble
export { ToolThoughtChain }
