import { useCallback, useMemo, useState } from 'react'
import { ThoughtChain } from '@ant-design/x'
import {
  buildToolTraceItems,
  renderMarkdown,
} from '../utils/helpers.jsx'

const ToolThoughtChain = ({ record = null, toolTraceRecords = [], toolDisplayNames }) => {
  const items = buildToolTraceItems({ record, toolTraceRecords, toolDisplayNames })
  const defaultExpandedKeys = useMemo(() => items
    .map((item) => item?.key)
    .filter(Boolean), [items])
  const [collapsedKeys, setCollapsedKeys] = useState([])
  const expandedKeys = useMemo(() => defaultExpandedKeys.filter((key) => !collapsedKeys.includes(key)), [collapsedKeys, defaultExpandedKeys])
  const handleExpand = useCallback((nextExpandedKeys) => {
    const expandedKeySet = new Set(nextExpandedKeys)
    setCollapsedKeys(defaultExpandedKeys.filter((key) => !expandedKeySet.has(key)))
  }, [defaultExpandedKeys])

  if (!items.length) {
    return null
  }

  return (
    <ThoughtChain
      className="assistant-thought-chain"
      items={items}
      expandedKeys={expandedKeys}
      onExpand={handleExpand}
      line="dashed"
    />
  )
}

const AssistantAnswerContent = ({ record }) => {
  const message = record.message || {}
  const rawAnswerContent = message.content || ''
  return renderMarkdown(rawAnswerContent)
}

export { AssistantAnswerContent, ToolThoughtChain }
