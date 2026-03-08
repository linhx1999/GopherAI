import { Pagination } from 'antd'
import { Welcome, Bubble } from '@ant-design/x'
import StreamBubble from './MessageBubble'
import { MESSAGE_PAGE_SIZE, MESSAGE_ROLES } from '../config/constants'

/**
 * 消息列表容器（含分页）
 */
const MessageList = ({
  messages,
  currentPage,
  roleConfig,
  onActionClick,
  onReasoningDisplayComplete,
  onPageChange
}) => {
  // 分页后的消息
  const start = (currentPage - 1) * MESSAGE_PAGE_SIZE
  const paginatedMessages = messages.slice(start, start + MESSAGE_PAGE_SIZE)

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
            // AI 消息使用 StreamBubble 组件
            if (item.role === MESSAGE_ROLES.AI) {
              return (
                <StreamBubble
                  key={item.key}
                  item={item}
                  onActionClick={onActionClick}
                  onReasoningDisplayComplete={onReasoningDisplayComplete}
                />
              )
            }
            // 其他消息使用标准 Bubble 组件
            const config = roleConfig[item.role] || roleConfig.user
            return (
              <Bubble
                key={item.key}
                {...config}
                content={item.content}
                loading={item.loading}
              />
            )
          })}
        </div>
      </div>

      {messages.length > MESSAGE_PAGE_SIZE && (
        <div className="pagination-container">
          <Pagination
            simple
            current={currentPage}
            onChange={onPageChange}
            total={messages.length}
            pageSize={MESSAGE_PAGE_SIZE}
            showSizeChanger={false}
          />
        </div>
      )}
    </>
  )
}

export default MessageList
