import { useMemo } from 'react'
import { Avatar, Pagination, Typography } from 'antd'
import { RobotOutlined } from '@ant-design/icons'
import { Welcome, Bubble, Actions } from '@ant-design/x'
import { AssistantAnswerContent, ToolThoughtChain } from './MessageBubble'
import { COLORS, MESSAGE_MAX_WIDTH, MESSAGE_PAGE_SIZE } from '../config/constants'
import { buildDisplayMessages, createMessageActions, renderMarkdown } from '../utils/helpers.jsx'

const { Text } = Typography

/**
 * 消息列表容器（含分页）
 */
const MessageList = ({
  messages,
  currentPage,
  toolDisplayNames,
  onActionClick,
  onPageChange,
  bubbleListRef,
  onListScroll
}) => {
  const displayMessages = buildDisplayMessages(messages)
  const start = (currentPage - 1) * MESSAGE_PAGE_SIZE
  const paginatedMessages = displayMessages.slice(start, start + MESSAGE_PAGE_SIZE)
  const bubbleItems = useMemo(() => paginatedMessages.map((item) => {
    if (item.role === 'thought_chain') {
      return {
        key: item.key,
        role: 'thought_chain',
        content: (
          <ToolThoughtChain
            record={item.record}
            toolTraceRecords={item.toolTraceRecords}
            toolDisplayNames={toolDisplayNames}
          />
        ),
        extraInfo: { record: item.record }
      }
    }

    if (item.role === 'assistant') {
      return {
        key: item.key,
        role: 'assistant',
        content: <AssistantAnswerContent record={item.record} />,
        extraInfo: { record: item.record }
      }
    }

    return {
      key: item.key,
      role: item.role,
      content: item.record.message?.content || '',
      loading: item.record.pending,
      extraInfo: { record: item.record }
    }
  }), [paginatedMessages, toolDisplayNames])
  const bubbleRoleConfig = useMemo(() => ({
    user: {
      placement: 'end',
      typing: false,
      avatar: <Avatar style={{ backgroundColor: COLORS.success }}>我</Avatar>,
      header: <Text type="secondary" style={{ fontSize: 12 }}>我</Text>,
      contentRender: renderMarkdown,
      footer: (_, info) => {
        const record = info.extraInfo?.record
        return record && !record.pending
          ? <Actions items={createMessageActions()} onClick={(actionInfo) => onActionClick(record, actionInfo)} />
          : null
      },
      style: { maxWidth: MESSAGE_MAX_WIDTH }
    },
    assistant: {
      placement: 'start',
      typing: false,
      avatar: <Avatar icon={<RobotOutlined />} style={{ backgroundColor: COLORS.primary }} />,
      header: <Text type="secondary" style={{ fontSize: 12 }}>AI 助手</Text>,
      contentRender: (content) => content,
      footer: (_, info) => {
        const record = info.extraInfo?.record
        return record && !record.pending
          ? <Actions items={createMessageActions()} onClick={(actionInfo) => onActionClick(record, actionInfo)} />
          : null
      },
      style: { maxWidth: MESSAGE_MAX_WIDTH }
    },
    thought_chain: {
      placement: 'start',
      variant: 'borderless',
      avatar: null,
      header: null,
      contentRender: (content) => content,
      styles: {
        root: { maxWidth: MESSAGE_MAX_WIDTH },
        body: { padding: 0 },
        content: {
          padding: 0,
          background: 'transparent',
          boxShadow: 'none'
        }
      },
      className: 'assistant-thought-chain-bubble'
    },
    system: {
      placement: 'start',
      variant: 'borderless',
      avatar: null,
      contentRender: renderMarkdown,
      style: { textAlign: 'center' }
    }
  }), [onActionClick])

  if (messages.length === 0) {
    return (
      <div className="messages-container">
        <Welcome title="Hello! 我是你的智能助手，请问有什么我可以帮助你的吗？" />
      </div>
    )
  }

  return (
    <>
      <div className="messages-container">
        <Bubble.List
          ref={bubbleListRef}
          className="message-list-bubble-list"
          items={bubbleItems}
          role={bubbleRoleConfig}
          autoScroll={false}
          onScroll={onListScroll}
        />
      </div>

      {displayMessages.length > MESSAGE_PAGE_SIZE && (
        <div className="pagination-container">
          <Pagination
            simple
            current={currentPage}
            onChange={onPageChange}
            total={displayMessages.length}
            pageSize={MESSAGE_PAGE_SIZE}
            showSizeChanger={false}
          />
        </div>
      )}
    </>
  )
}

export default MessageList
