import { useRef, useMemo } from 'react'
import { Button, Checkbox, Dropdown, Flex, Segmented, Space, Tag, Tooltip } from 'antd'
import {
  ApiOutlined,
  CloudUploadOutlined,
  DeploymentUnitOutlined,
  LinkOutlined,
  ReloadOutlined,
  ToolOutlined,
  ThunderboltOutlined,
  SearchOutlined,
  BulbOutlined,
  SyncOutlined
} from '@ant-design/icons'
import { Sender, Attachments } from '@ant-design/x'
import { AGENT_MODES } from '../config/constants'

const Switch = Sender.Switch

const resolveToolIcon = (tool) => {
  if (tool?.apiName === 'knowledge_search') {
    return SearchOutlined
  }
  if (tool?.apiName === 'sequentialthinking') {
    return BulbOutlined
  }
  return ToolOutlined
}

const RUNTIME_STATUS_META = {
  stopped: { color: 'default', label: '已停止' },
  starting: { color: 'processing', label: '启动中' },
  running: { color: 'success', label: '运行中' },
  error: { color: 'error', label: '异常' },
  rebuilding: { color: 'warning', label: '重建中' }
}

/**
 * 输入区域组件（含工具选择和附件）
 */
const InputArea = ({
  inputValue,
  isLoading,
  availableTools,
  availableMCPServers,
  deepAgentEnabled,
  enabledToolApiNames,
  enabledMCPServerIDs,
  agentMode,
  thinkingMode,
  isStreaming,
  deepAgentRuntime,
  deepAgentRuntimeLoading,
  attachments,
  attachmentsOpen,
  onInputChange,
  onSubmit,
  onAgentModeChange,
  onEnabledToolApiNamesChange,
  onEnabledMCPServerIDsChange,
  onThinkingModeChange,
  onStreamingChange,
  onDeepAgentRuntimeRefresh,
  onDeepAgentRuntimeRestart,
  onDeepAgentRuntimeRebuild,
  onAttachmentsChange,
  onAttachmentsOpenChange
}) => {
  const senderRef = useRef(null)
  const runtimeMeta = RUNTIME_STATUS_META[deepAgentRuntime?.status] || RUNTIME_STATUS_META.stopped
  const isDeepMode = agentMode === AGENT_MODES.DEEP

  const isServerSelectable = (server) => (
    server?.lastTestStatus === 'success' && Array.isArray(server?.tools) && server.tools.length > 0
  )

  const getServerHint = (server) => {
    if (!server) {
      return ''
    }
    if (server.lastTestStatus !== 'success') {
      return server.lastTestMessage || '请先在 MCP 管理页完成连接测试'
    }
    if (!server.tools?.length) {
      return '该服务当前没有可用工具'
    }
    return `${server.tools.length} 个工具`
  }

  // 附件区域头部
  const senderHeader = useMemo(() => (
    <Sender.Header
      title="附件"
      open={attachmentsOpen}
      onOpenChange={onAttachmentsOpenChange}
      styles={{
        content: {
          padding: 0,
        },
      }}
    >
      <Attachments
        beforeUpload={() => false}
        items={attachments}
        onChange={({ file, fileList }) => {
          const updatedFileList = fileList.map(item => {
            if (item.uid === file.uid && file.status !== 'removed' && item.originFileObj) {
              // 清理旧 URL
              if (item.url?.startsWith('blob:')) {
                URL.revokeObjectURL(item.url)
              }
              // 创建预览 URL
              return {
                ...item,
                url: URL.createObjectURL(item.originFileObj)
              }
            }
            return item
          })
          onAttachmentsChange(updatedFileList)
        }}
        placeholder={type =>
          type === 'drop'
            ? {
                title: '拖放文件到此处',
              }
            : {
                icon: <CloudUploadOutlined />,
                title: '上传文件',
                description: '点击或拖拽文件到此区域上传',
              }
        }
        getDropContainer={() => senderRef.current?.nativeElement}
      />
    </Sender.Header>
  ), [attachmentsOpen, attachments, onAttachmentsChange, onAttachmentsOpenChange])

  return (
    <div className="input-container">
      <Sender
        ref={senderRef}
        header={senderHeader}
        value={inputValue}
        onChange={onInputChange}
        onSubmit={onSubmit}
        loading={isLoading}
        suffix={false}
        placeholder="请输入你的问题..."
        footer={(actionNode) => (
          <Flex justify="space-between" align="center">
            {/* 左侧控制区 */}
            <Flex gap="small" align="center">
              <Segmented
                size="small"
                value={agentMode}
                onChange={onAgentModeChange}
                options={[
                  {
                    label: <Space size={4}><ToolOutlined />普通聊天</Space>,
                    value: AGENT_MODES.CHAT
                  },
                  {
                    label: <Space size={4}><DeploymentUnitOutlined />DeepAgent</Space>,
                    value: AGENT_MODES.DEEP,
                    disabled: !deepAgentEnabled
                  }
                ]}
              />

              {/* 附件按钮 */}
              <Button
                type="text"
                icon={<LinkOutlined />}
                onClick={() => onAttachmentsOpenChange(!attachmentsOpen)}
              />

              {/* 流式响应开关 */}
              <Switch
                value={isStreaming}
                checkedChildren="流式"
                unCheckedChildren="普通"
                onChange={onStreamingChange}
                icon={<ThunderboltOutlined />}
              />

              <Switch
                value={thinkingMode}
                onChange={onThinkingModeChange}
                icon={<BulbOutlined />}
              >
                思考
              </Switch>

              {/* 工具选择下拉菜单 */}
              <Dropdown
                trigger={['click']}
                popupRender={() => (
                  <div style={{
                    padding: 12,
                    background: '#fff',
                    borderRadius: 8,
                    boxShadow: '0 2px 8px rgba(0,0,0,0.15)',
                    minWidth: 160
                  }}>
                    <div style={{ marginBottom: 8, fontWeight: 500, color: '#666' }}>
                      选择工具
                    </div>
                    <Checkbox.Group
                      value={enabledToolApiNames}
                      onChange={onEnabledToolApiNamesChange}
                      style={{ display: 'flex', flexDirection: 'column', gap: 8 }}
                    >
                      {availableTools.map((tool) => {
                        const IconComponent = resolveToolIcon(tool)
                        return (
                          <Checkbox key={tool.apiName} value={tool.apiName}>
                            <Flex align="center" gap={4}>
                              <IconComponent />
                              {tool.displayName}
                            </Flex>
                          </Checkbox>
                        )
                      })}
                    </Checkbox.Group>
                    {availableTools.length === 0 ? (
                      <div style={{ marginTop: 8, color: '#999', fontSize: 12 }}>
                        当前没有可用工具
                      </div>
                    ) : null}
                  </div>
                )}
              >
                <Switch value={enabledToolApiNames.length > 0} icon={<ToolOutlined />}>
                  工具 {enabledToolApiNames.length > 0 && `(${enabledToolApiNames.length})`}
                </Switch>
              </Dropdown>

              <Dropdown
                trigger={['click']}
                popupRender={() => (
                  <div style={{
                    padding: 12,
                    background: '#fff',
                    borderRadius: 8,
                    boxShadow: '0 2px 8px rgba(0,0,0,0.15)',
                    minWidth: 240
                  }}>
                    <div style={{ marginBottom: 8, fontWeight: 500, color: '#666' }}>
                      选择 MCP 服务
                    </div>
                    <Checkbox.Group
                      value={enabledMCPServerIDs}
                      onChange={onEnabledMCPServerIDsChange}
                      style={{ display: 'flex', flexDirection: 'column', gap: 10 }}
                    >
                      {availableMCPServers.map((server) => {
                        const disabled = !isServerSelectable(server)
                        return (
                          <Checkbox key={server.serverId} value={server.serverId} disabled={disabled}>
                            <Flex vertical gap={2}>
                              <Flex align="center" gap={6}>
                                <ApiOutlined />
                                <span>{server.name}</span>
                              </Flex>
                              <span style={{ color: '#999', fontSize: 12 }}>
                                {getServerHint(server)}
                              </span>
                            </Flex>
                          </Checkbox>
                        )
                      })}
                    </Checkbox.Group>
                    {availableMCPServers.length === 0 ? (
                      <div style={{ marginTop: 8, color: '#999', fontSize: 12 }}>
                        当前没有已配置的 MCP 服务
                      </div>
                    ) : null}
                  </div>
                )}
              >
                <Switch value={enabledMCPServerIDs.length > 0} icon={<ApiOutlined />}>
                  MCP {enabledMCPServerIDs.length > 0 && `(${enabledMCPServerIDs.length})`}
                </Switch>
              </Dropdown>

            </Flex>

            {/* 右侧提交区 */}
            <Flex align="center">
              {actionNode}
            </Flex>
          </Flex>
        )}
      />

      {isDeepMode ? (
        <div style={{
          marginTop: 8,
          padding: '10px 12px',
          borderRadius: 12,
          background: '#f7f8fa',
          border: '1px solid #eef1f5'
        }}>
          <Flex justify="space-between" align="center" gap={12} wrap>
            <Space size="small" wrap>
              <Tag color={runtimeMeta.color}>{runtimeMeta.label}</Tag>
              <span style={{ color: '#666', fontSize: 12 }}>
                {deepAgentRuntime?.lastError || '每个用户复用一个独立容器与空工作区'}
              </span>
            </Space>

            <Space size="small" wrap>
              <Tooltip title="刷新运行时状态">
                <Button
                  size="small"
                  icon={<SyncOutlined spin={deepAgentRuntimeLoading} />}
                  onClick={onDeepAgentRuntimeRefresh}
                  disabled={deepAgentRuntimeLoading}
                >
                  刷新
                </Button>
              </Tooltip>
              <Button
                size="small"
                icon={<ReloadOutlined />}
                onClick={onDeepAgentRuntimeRestart}
                disabled={deepAgentRuntimeLoading}
              >
                重启容器
              </Button>
              <Button
                size="small"
                onClick={onDeepAgentRuntimeRebuild}
                disabled={deepAgentRuntimeLoading}
              >
                重建工作区
              </Button>
            </Space>
          </Flex>
        </div>
      ) : null}
    </div>
  )
}

export default InputArea
