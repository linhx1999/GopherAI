import { Avatar, Collapse, Tag, Typography } from 'antd'
import { RobotOutlined, ToolOutlined } from '@ant-design/icons'
import { Bubble, Actions, Think } from '@ant-design/x'
import useStreamContent from '../hooks/useStreamContent'
import { COLORS, MESSAGE_MAX_WIDTH } from '../config/constants'
import { createMessageActions, formatToolArguments, renderMarkdown } from '../utils/helpers.jsx'

const { Text, Paragraph } = Typography

const ProcessCard = ({ processes = [] }) => {
  if (!processes.length) return null

  const items = processes.map((record) => {
    const message = record.message || {}
    const toolCalls = message.tool_calls || []
    const isToolMessage = message.role === 'tool'
    const toolTitle = isToolMessage ? (message.tool_name || '工具结果') : '工具调用'

    return {
      key: record.key,
      label: (
        <span className="process-item-title">
          <ToolOutlined />
          <span>{toolTitle}</span>
          {record.pending ? <Tag color="processing">执行中</Tag> : null}
        </span>
      ),
      children: (
        <div className="process-item-body">
          {toolCalls.map((call, index) => (
            <div key={`${record.key}-tool-${index}`} className="process-tool-call">
              <Text strong>{call.function?.name || call.id || 'tool'}</Text>
              <pre>{formatToolArguments(call.function?.arguments)}</pre>
            </div>
          ))}
          {message.content ? (
            <div className="process-tool-result">
              {renderMarkdown(message.content)}
            </div>
          ) : null}
          {!message.content && toolCalls.length === 0 ? (
            <Paragraph type="secondary" style={{ marginBottom: 0 }}>
              暂无可展示内容
            </Paragraph>
          ) : null}
        </div>
      )
    }
  })

  return (
    <Collapse
      className="assistant-process-card"
      defaultActiveKey={[]}
      items={[{
        key: 'processes',
        label: `执行过程 (${processes.length})`,
        children: <Collapse ghost items={items} />
      }]}
    />
  )
}

const AssistantBubble = ({ record, processes, onActionClick }) => {
  const message = record.message || {}
  const [streamContent, isContentDone] = useStreamContent(message.content || '', { step: 3, interval: 30 })
  const [streamReasoning, isReasoningDone] = useStreamContent(message.reasoning_content || '', { step: 3, interval: 30 })
  const reasoningDisplayContent = isReasoningDone ? (message.reasoning_content || '') : streamReasoning
  const answerDisplayContent = isContentDone ? (message.content || '') : streamContent
  const isStreaming = record.pending && !isContentDone
  const isReasoningStreaming = record.pending && !isReasoningDone
  const showReasoning = Boolean(message.reasoning_content)
  const showAnswer = Boolean(message.content)

  return (
    <div className="assistant-message">
      <ProcessCard processes={processes} />

      {showReasoning ? (
        <Think
          title="深度思考"
          loading={isReasoningStreaming}
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
export { ProcessCard }
