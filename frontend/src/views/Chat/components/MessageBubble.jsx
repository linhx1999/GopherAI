import { Avatar, Typography } from 'antd'
import { RobotOutlined } from '@ant-design/icons'
import { Actions, Bubble, ThoughtChain } from '@ant-design/x'
import useStreamContent from '../hooks/useStreamContent'
import { COLORS, MESSAGE_MAX_WIDTH } from '../config/constants'
import {
  buildToolTraceItems,
  createMessageActions,
  renderMarkdown,
  shouldRenderToolTrace
} from '../utils/helpers.jsx'

const { Text } = Typography

const ToolThoughtChain = ({ record = null, toolTraceRecords = [], toolDisplayNames }) => {
  const items = buildToolTraceItems({ record, toolTraceRecords, toolDisplayNames })
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

const AssistantBubble = ({ record, toolTraceRecords, toolDisplayNames, onActionClick }) => {
  const message = record.message || {}
  const shouldAnimate = record.renderMode === 'stream' && record.pending
  const useToolTrace = shouldRenderToolTrace(record, toolTraceRecords)
  const rawAnswerContent = message.content || ''
  const [streamContent, isContentDone] = useStreamContent(rawAnswerContent, {
    step: 3,
    interval: 30,
    enabled: shouldAnimate
  })
  const answerDisplayContent = shouldAnimate && !isContentDone ? streamContent : rawAnswerContent
  const isStreaming = shouldAnimate && !isContentDone
  const showAnswer = Boolean(rawAnswerContent)

  return (
    <div className="assistant-message">
      {useToolTrace ? (
        <ToolThoughtChain
          record={record}
          toolTraceRecords={toolTraceRecords}
          toolDisplayNames={toolDisplayNames}
        />
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
