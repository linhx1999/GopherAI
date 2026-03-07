import { useMemo } from 'react'
import { Layout } from 'antd'
import SessionSider from './components/SessionSider'
import ChatHeader from './components/ChatHeader'
import MessageList from './components/MessageList'
import InputArea from './components/InputArea'
import useChat from './hooks/useChat'
import { createRoleConfig } from './utils/helpers.jsx'
import './index.css'

const { Sider, Content } = Layout

/**
 * 聊天页面主布局
 */
const Chat = () => {
  const {
    // refs
    bubbleListRef,
    // 基础状态
    selectedTools,
    isStreaming,
    currentPage,
    inputValue,
    isLoading,
    attachments,
    attachmentsOpen,
    messages,
    sessions,
    activeKey,
    isTempSession,
    editingSession,
    editTitle,
    // 会话操作
    createSession,
    switchSession,
    handleMenuClick,
    confirmRename,
    setEditTitle,
    // 消息操作
    handleSend,
    handleActionClick,
    // 状态更新
    setSelectedTools,
    setIsStreaming,
    setCurrentPage,
    setInputValue,
    setAttachments,
    setAttachmentsOpen
  } = useChat()

  // Role 配置
  const roleConfig = useMemo(() => createRoleConfig(handleActionClick), [handleActionClick])

  return (
    <Layout>
      <Sider>
        <SessionSider
          sessions={sessions}
          activeKey={activeKey}
          editingSession={editingSession}
          editTitle={editTitle}
          onCreateSession={createSession}
          onSwitchSession={switchSession}
          onMenuClick={handleMenuClick}
          onEditTitleChange={setEditTitle}
          onConfirmRename={confirmRename}
        />
      </Sider>

      <Content className="chat-content">
        <ChatHeader />

        <MessageList
          messages={messages}
          currentPage={currentPage}
          roleConfig={roleConfig}
          onActionClick={handleActionClick}
          onPageChange={setCurrentPage}
        />

        <InputArea
          inputValue={inputValue}
          isLoading={isLoading}
          selectedTools={selectedTools}
          isStreaming={isStreaming}
          attachments={attachments}
          attachmentsOpen={attachmentsOpen}
          onInputChange={setInputValue}
          onSubmit={handleSend}
          onSelectedToolsChange={setSelectedTools}
          onStreamingChange={setIsStreaming}
          onAttachmentsChange={setAttachments}
          onAttachmentsOpenChange={setAttachmentsOpen}
        />
      </Content>
    </Layout>
  )
}

export default Chat
