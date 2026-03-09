import { Pagination } from 'antd'
import { Welcome, Bubble, Actions } from '@ant-design/x'
import AssistantBubble, { ToolThoughtChain } from './MessageBubble'
import { MESSAGE_PAGE_SIZE, MESSAGE_ROLES } from '../config/constants'
import { buildDisplayMessages, createMessageActions } from '../utils/helpers.jsx'

/**
 * 消息列表容器（含分页）
 */
const MessageList = ({
  messages,
  currentPage,
  roleConfig,
  toolDisplayNames,
  onActionClick,
  onPageChange
}) => {
  const displayMessages = buildDisplayMessages(messages)
  const start = (currentPage - 1) * MESSAGE_PAGE_SIZE
  const paginatedMessages = displayMessages.slice(start, start + MESSAGE_PAGE_SIZE)

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
        <div className="message-list-container">
          {paginatedMessages.map((item) => {
            if (item.type === 'assistant') {
              return (
                <AssistantBubble
                  key={item.key}
                  record={item.record}
                  toolTraceRecords={item.toolTraceRecords}
                  toolDisplayNames={toolDisplayNames}
                  onActionClick={onActionClick}
                />
              )
            }

            if (item.type === 'tool_trace') {
              return (
                <ToolThoughtChain
                  key={item.key}
                  toolTraceRecords={item.toolTraceRecords}
                  toolDisplayNames={toolDisplayNames}
                />
              )
            }

            const record = item.record
            const role = record.message?.role === MESSAGE_ROLES.SYSTEM ? MESSAGE_ROLES.SYSTEM : MESSAGE_ROLES.USER
            const config = roleConfig[role] || roleConfig.user
            return (
              <Bubble
                key={record.key}
                {...config}
                content={record.message?.content}
                loading={record.pending}
                footer={role === MESSAGE_ROLES.USER && !record.pending ? (
                  <Actions items={createMessageActions(true)} onClick={(info) => onActionClick(record, info)} />
                ) : null}
              />
            )
          })}
        </div>
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
