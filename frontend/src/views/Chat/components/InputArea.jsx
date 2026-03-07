import { useRef, useMemo } from 'react'
import { Button, Checkbox, Dropdown, Flex } from 'antd'
import {
  CloudUploadOutlined,
  LinkOutlined,
  ToolOutlined,
  ThunderboltOutlined,
  SearchOutlined,
  BulbOutlined,
  CloudOutlined
} from '@ant-design/icons'
import { Sender, Attachments } from '@ant-design/x'
import { TOOL_OPTIONS } from '../config/constants'

const Switch = Sender.Switch

/**
 * 输入区域组件（含工具选择和附件）
 */
const InputArea = ({
  inputValue,
  isLoading,
  selectedTools,
  isStreaming,
  attachments,
  attachmentsOpen,
  onInputChange,
  onSubmit,
  onSelectedToolsChange,
  onStreamingChange,
  onAttachmentsChange,
  onAttachmentsOpenChange
}) => {
  const senderRef = useRef(null)

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

              {/* 工具选择下拉菜单 */}
              <Dropdown
                trigger={['click']}
                dropdownRender={() => (
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
                      value={selectedTools}
                      onChange={onSelectedToolsChange}
                      style={{ display: 'flex', flexDirection: 'column', gap: 8 }}
                    >
                      {TOOL_OPTIONS.map(tool => {
                        const IconComponent = tool.icon === 'SearchOutlined' ? SearchOutlined :
                          tool.icon === 'BulbOutlined' ? BulbOutlined : CloudOutlined
                        return (
                          <Checkbox key={tool.value} value={tool.value}>
                            <Flex align="center" gap={4}>
                              <IconComponent />
                              {tool.label}
                            </Flex>
                          </Checkbox>
                        )
                      })}
                    </Checkbox.Group>
                  </div>
                )}
              >
                <Switch value={selectedTools.length > 0} icon={<ToolOutlined />}>
                  工具 {selectedTools.length > 0 && `(${selectedTools.length})`}
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
    </div>
  )
}

export default InputArea
