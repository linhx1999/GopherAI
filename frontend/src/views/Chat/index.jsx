import { useMemo } from 'react'
import { Layout } from 'antd'
import SessionSider from './components/SessionSider'
import ChatHeader from './components/ChatHeader'
import MessageList from './components/MessageList'
import InputArea from './components/InputArea'
import useChat from './hooks/useChat'
import { createToolDisplayNameMap } from './utils/helpers.jsx'
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
    availableMCPServers,
    mcpFeatureEnabled,
    deepAgentEnabled,
    enabledToolAPINames,
    enabledMCPServerIDs,
    agentMode,
    thinkingMode,
    isStreaming,
    deepAgentRuntime,
    deepAgentRuntimeLoading,
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
    bubbleListRef,
    handleBubbleListScroll,
    refreshDeepAgentRuntime,
    restartDeepAgentRuntime,
    rebuildDeepAgentRuntime,
    // 状态更新
    setAgentMode,
    setEnabledToolAPINames,
    setEnabledMCPServerIDs,
    setThinkingMode,
    setIsStreaming,
    setCurrentPage,
    setInputValue,
    setAttachments,
    setAttachmentsOpen
  } = useChat()

  const toolDisplayNames = useMemo(() => createToolDisplayNameMap([
    ...availableTools,
    ...availableMCPServers.flatMap((server) => server.tools || [])
  ]), [availableMCPServers, availableTools])

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
          toolDisplayNames={toolDisplayNames}
          onActionClick={handleActionClick}
          onPageChange={setCurrentPage}
          bubbleListRef={bubbleListRef}
          onListScroll={handleBubbleListScroll}
        />

        <InputArea
          inputValue={inputValue}
          isLoading={isLoading}
          availableTools={availableTools}
          availableMCPServers={mcpFeatureEnabled ? availableMCPServers : []}
          deepAgentEnabled={deepAgentEnabled}
          enabledToolApiNames={enabledToolAPINames}
          enabledMCPServerIDs={enabledMCPServerIDs}
          agentMode={agentMode}
          thinkingMode={thinkingMode}
          isStreaming={isStreaming}
          deepAgentRuntime={deepAgentRuntime}
          deepAgentRuntimeLoading={deepAgentRuntimeLoading}
          attachments={attachments}
          attachmentsOpen={attachmentsOpen}
          onInputChange={setInputValue}
          onSubmit={handleSend}
          onAgentModeChange={setAgentMode}
          onEnabledToolApiNamesChange={setEnabledToolAPINames}
          onEnabledMCPServerIDsChange={setEnabledMCPServerIDs}
          onThinkingModeChange={setThinkingMode}
          onStreamingChange={setIsStreaming}
          onDeepAgentRuntimeRefresh={refreshDeepAgentRuntime}
          onDeepAgentRuntimeRestart={restartDeepAgentRuntime}
          onDeepAgentRuntimeRebuild={rebuildDeepAgentRuntime}
          onAttachmentsChange={setAttachments}
          onAttachmentsOpenChange={setAttachmentsOpen}
        />
      </Content>
    </Layout>
  )
}

export default Chat
