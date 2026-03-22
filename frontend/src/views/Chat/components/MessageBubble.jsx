import { ThoughtChain } from '@ant-design/x'
import useStreamContent from '../hooks/useStreamContent'
import {
  buildToolTraceItems,
  renderMarkdown,
} from '../utils/helpers.jsx'

const ToolThoughtChain = ({ record = null, toolTraceRecords = [], toolDisplayNames }) => {
  const items = buildToolTraceItems({ record, toolTraceRecords, toolDisplayNames })
  const defaultExpandedKeys = items
    .map((item) => item?.key)
    .filter(Boolean)
  const thoughtChainKey = defaultExpandedKeys.join('|')

  if (!items.length) {
    return null
  }

  return (
    <ThoughtChain
      key={thoughtChainKey}
      className="assistant-thought-chain"
      items={items}
      defaultExpandedKeys={defaultExpandedKeys}
      line="dashed"
    />
  )
}

const AssistantAnswerContent = ({ record }) => {
  const message = record.message || {}
  const shouldAnimate = record.renderMode === 'stream' && record.pending
  const rawAnswerContent = message.content || ''
  const [streamContent, isContentDone] = useStreamContent(rawAnswerContent, {
    step: 3,
    interval: 30,
    enabled: shouldAnimate
  })
  const answerDisplayContent = shouldAnimate && !isContentDone ? streamContent : rawAnswerContent
  return renderMarkdown(answerDisplayContent)
}

export { AssistantAnswerContent, ToolThoughtChain }
