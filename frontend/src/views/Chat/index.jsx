import { useMemo } from 'react'
import { Layout } from 'antd'
import SessionSider from './components/SessionSider'
import ChatHeader from './components/ChatHeader'
import MessageList from './components/MessageList'
import InputArea from './components/InputArea'
import useChat from './hooks/useChat'
import { createRoleConfig, createToolDisplayNameMap } from './utils/helpers.jsx'
import './index.css'
import { theme } from 'antd';

const { Sider, Content } = Layout

/**
 * 聊天页面主布局
 */
const Chat = () => {
  const { token } = theme.useToken();
  const style = {
    height: '100vh',
    background: token.colorBgContainer,
    borderRadius: token.borderRadius,
  };
  const {
    // 基础状态
    availableTools,
    enabledToolAPINames,
    thinkingMode,
    isStreaming,
    currentPage,
    inputValue,
    isLoading,
    attachments,
    attachmentsOpen,
    messages,
    sessions,
    activeKey,
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
    setEnabledToolAPINames,
    setThinkingMode,
    setIsStreaming,
    setCurrentPage,
    setInputValue,
    setAttachments,
    setAttachmentsOpen
  } = useChat()

  // Role 配置
  const roleConfig = useMemo(() => createRoleConfig(), [])
  const toolDisplayNames = useMemo(() => createToolDisplayNameMap(availableTools), [availableTools])

  return (
    <Layout style={style}>
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
          toolDisplayNames={toolDisplayNames}
          onActionClick={handleActionClick}
          onPageChange={setCurrentPage}
        />

        <InputArea
          inputValue={inputValue}
          isLoading={isLoading}
          availableTools={availableTools}
          enabledToolApiNames={enabledToolAPINames}
          thinkingMode={thinkingMode}
          isStreaming={isStreaming}
          attachments={attachments}
          attachmentsOpen={attachmentsOpen}
          onInputChange={setInputValue}
          onSubmit={handleSend}
          onEnabledToolApiNamesChange={setEnabledToolAPINames}
          onThinkingModeChange={setThinkingMode}
          onStreamingChange={setIsStreaming}
          onAttachmentsChange={setAttachments}
          onAttachmentsOpenChange={setAttachmentsOpen}
        />
      </Content>
    </Layout>
  )
}

export default Chat
