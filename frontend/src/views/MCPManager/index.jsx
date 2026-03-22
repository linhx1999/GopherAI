import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Alert,
  App,
  Button,
  Drawer,
  Empty,
  Form,
  Input,
  Layout,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  theme
} from 'antd'
import {
  ApiOutlined,
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  LogoutOutlined,
  PlusOutlined,
  ReloadOutlined,
  RollbackOutlined
} from '@ant-design/icons'
import useMCPManager from './hooks/useMCPManager'
import './index.css'

const { Content } = Layout

const STATUS_META = {
  success: { color: 'success', label: '连接正常' },
  failed: { color: 'error', label: '连接失败' },
  untested: { color: 'default', label: '未测试' }
}

const hasSavedHeaderValue = (value) => value === true || value === 'true'

const buildHeaderPayload = (headerItems = []) => (
  headerItems
    .map((item) => ({
      key: String(item?.key || '').trim(),
      value: String(item?.value || '').trim(),
      hasValue: hasSavedHeaderValue(item?.hasValue),
    }))
    .filter((item) => item.key)
    .map((item) => ({
      key: item.key,
      value: item.value,
      keep_existing: item.hasValue && !item.value
    }))
)

const MCPManager = () => {
  const navigate = useNavigate()
  const { message } = App.useApp()
  const { token } = theme.useToken()
  const [form] = Form.useForm()
  const {
    servers,
    loading,
    featureDisabled,
    errorMessage,
    loadServers,
    fetchServerDetail,
    createServer,
    updateServer,
    deleteServer,
    testServer
  } = useMCPManager()

  const [drawerOpen, setDrawerOpen] = useState(false)
  const [drawerLoading, setDrawerLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [editingServerId, setEditingServerId] = useState(null)
  const [drawerTools, setDrawerTools] = useState([])
  const [previewServer, setPreviewServer] = useState(null)

  const pageStyle = {
    minHeight: '100vh',
    background: token.colorBgLayout,
  }

  const summary = useMemo(() => ({
    total: servers.length,
    success: servers.filter((server) => server.lastTestStatus === 'success').length,
    failed: servers.filter((server) => server.lastTestStatus === 'failed').length,
    tools: servers.reduce((count, server) => count + (server.tools?.length || 0), 0)
  }), [servers])

  const handleLogout = () => {
    Modal.confirm({
      title: '提示',
      content: '确定要退出登录吗？',
      okText: '确定',
      cancelText: '取消',
      onOk: () => {
        localStorage.removeItem('token')
        message.success('退出登录成功')
        navigate('/login')
      }
    })
  }

  const resetDrawerState = () => {
    setEditingServerId(null)
    setDrawerTools([])
    form.resetFields()
  }

  const openCreateDrawer = () => {
    resetDrawerState()
    form.setFieldsValue({
      name: '',
      transportType: 'sse',
      endpoint: '',
      headers: []
    })
    setDrawerOpen(true)
  }

  const openEditDrawer = async (serverId) => {
    setDrawerLoading(true)
    setDrawerOpen(true)
    try {
      const detail = await fetchServerDetail(serverId)
      setEditingServerId(serverId)
      setDrawerTools(detail.tools || [])
      form.setFieldsValue({
        name: detail.name,
        transportType: detail.transportType || 'sse',
        endpoint: detail.endpoint,
        headers: (detail.headers || []).map((item) => ({
          key: item.key,
          value: '',
          hasValue: item.hasValue,
          maskedValue: item.maskedValue
        }))
      })
    } catch (error) {
      message.error(error.message)
      setDrawerOpen(false)
    } finally {
      setDrawerLoading(false)
    }
  }

  const handleDrawerClose = () => {
    setDrawerOpen(false)
    resetDrawerState()
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      const payload = {
        name: values.name,
        endpoint: values.endpoint,
        transport_type: values.transportType,
        headers: buildHeaderPayload(values.headers)
      }

      setSaving(true)
      if (editingServerId) {
        await updateServer(editingServerId, payload)
        message.success('MCP 服务已更新')
      } else {
        await createServer(payload)
        message.success('MCP 服务已创建')
      }

      handleDrawerClose()
      await loadServers()
    } catch (error) {
      if (error?.errorFields) {
        return
      }
      message.error(error.message || '保存 MCP 服务失败')
    } finally {
      setSaving(false)
    }
  }

  const handleQuickTest = async (serverId) => {
    try {
      const server = await testServer(serverId)
      message.success(`测试成功，发现 ${server.tools.length} 个工具`)
      await loadServers()
    } catch (error) {
      message.error(error.message || '测试连接失败')
      await loadServers()
    }
  }

  const handleDrawerTest = async () => {
    if (!editingServerId) {
      message.warning('请先保存 MCP 服务，再执行连接测试')
      return
    }

    setTesting(true)
    try {
      const server = await testServer(editingServerId)
      setDrawerTools(server.tools || [])
      message.success(`测试成功，发现 ${server.tools.length} 个工具`)
      await loadServers()
    } catch (error) {
      message.error(error.message || '测试连接失败')
      await loadServers()
    } finally {
      setTesting(false)
    }
  }

  const handleDelete = (serverId) => {
    Modal.confirm({
      title: '删除 MCP 服务',
      content: '删除后聊天页将无法继续启用该服务，确定继续吗？',
      okText: '删除',
      okButtonProps: { danger: true },
      cancelText: '取消',
      onOk: async () => {
        try {
          await deleteServer(serverId)
          message.success('MCP 服务已删除')
          await loadServers()
        } catch (error) {
          message.error(error.message || '删除 MCP 服务失败')
        }
      }
    })
  }

  const handlePreview = async (serverId) => {
    try {
      const detail = await fetchServerDetail(serverId)
      setPreviewServer(detail)
    } catch (error) {
      message.error(error.message || '获取工具预览失败')
    }
  }

  const columns = [
    {
      title: '服务',
      dataIndex: 'name',
      key: 'name',
      render: (_, record) => (
        <div className="mcp-manager-name">
          <strong>{record.name || '未命名服务'}</strong>
          <span>{record.transportType === 'http' ? 'HTTP' : 'SSE'}</span>
        </div>
      )
    },
    {
      title: 'Endpoint',
      dataIndex: 'endpoint',
      key: 'endpoint',
      render: (value) => <span className="mcp-manager-endpoint">{value}</span>
    },
    {
      title: '状态',
      dataIndex: 'lastTestStatus',
      key: 'lastTestStatus',
      render: (_, record) => {
        const meta = STATUS_META[record.lastTestStatus] || STATUS_META.untested
        return (
          <Space orientation="vertical" size={4}>
            <Tag color={meta.color}>{meta.label}</Tag>
            {record.lastTestMessage ? (
              <span className="mcp-manager-hint">{record.lastTestMessage}</span>
            ) : null}
          </Space>
        )
      }
    },
    {
      title: '工具数',
      dataIndex: 'tools',
      key: 'tools',
      width: 100,
      render: (tools = []) => tools.length
    },
    {
      title: '最近测试',
      dataIndex: 'lastTestedAt',
      key: 'lastTestedAt',
      render: (value) => value || '未测试'
    },
    {
      title: '操作',
      key: 'actions',
      width: 260,
      render: (_, record) => (
        <Space wrap>
          <Button size="small" icon={<EditOutlined />} onClick={() => void openEditDrawer(record.serverId)}>
            编辑
          </Button>
          <Button size="small" icon={<ReloadOutlined />} onClick={() => void handleQuickTest(record.serverId)}>
            测试
          </Button>
          <Button size="small" icon={<EyeOutlined />} onClick={() => void handlePreview(record.serverId)}>
            查看工具
          </Button>
          <Button
            size="small"
            danger
            icon={<DeleteOutlined />}
            onClick={() => handleDelete(record.serverId)}
          >
            删除
          </Button>
        </Space>
      )
    }
  ]

  return (
    <Layout style={pageStyle} className="mcp-manager-layout">
      <Content className="mcp-manager-content">
        <div className="mcp-manager-toolbar">
          <Space wrap>
            <Button icon={<RollbackOutlined />} onClick={() => navigate('/menu')}>
              返回
            </Button>
            <Button danger icon={<LogoutOutlined />} onClick={handleLogout}>
              退出登录
            </Button>
          </Space>

          <div className="mcp-manager-toolbar-copy">
            <span className="mcp-manager-eyebrow">Model Context Protocol</span>
            <h1>MCP 管理</h1>
            <p>配置远程 MCP 服务，测试连接并把整组服务能力接入聊天页。</p>
          </div>
        </div>

        <section className="mcp-manager-shell">
          <div className="mcp-manager-panel">
            <div className="mcp-manager-panel-toolbar">
              <div className="mcp-manager-panel-copy">
                <span className="mcp-manager-panel-eyebrow">Remote Tools</span>
                <h2>用户自定义 MCP 服务</h2>
                <p>当前支持基于 SSE 或 streamable HTTP 的远程 MCP。每次聊天按服务整体启用，服务下所有工具将一起加入该轮对话。</p>
              </div>

              <div className="mcp-manager-panel-actions">
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={openCreateDrawer}
                  disabled={featureDisabled}
                >
                  新增 MCP 服务
                </Button>

                <div className="mcp-manager-stats">
                  <div className="mcp-manager-stat-card">
                    <span className="mcp-manager-stat-label">服务总数</span>
                    <strong>{summary.total}</strong>
                  </div>
                  <div className="mcp-manager-stat-card">
                    <span className="mcp-manager-stat-label">连接正常</span>
                    <strong>{summary.success}</strong>
                  </div>
                  <div className="mcp-manager-stat-card">
                    <span className="mcp-manager-stat-label">连接失败</span>
                    <strong>{summary.failed}</strong>
                  </div>
                  <div className="mcp-manager-stat-card">
                    <span className="mcp-manager-stat-label">工具总数</span>
                    <strong>{summary.tools}</strong>
                  </div>
                </div>
              </div>
            </div>

            <div className="mcp-manager-table-panel">
              {featureDisabled ? (
                <Alert
                  type="warning"
                  showIcon
                  message="MCP 功能未启用"
                  description="请先在服务端配置 `MCP_SECRET_KEY`，用于加密保存自定义请求头。"
                  style={{ marginBottom: 16 }}
                />
              ) : null}

              {!featureDisabled && errorMessage ? (
                <Alert
                  type="error"
                  showIcon
                  message="加载失败"
                  description={errorMessage}
                  style={{ marginBottom: 16 }}
                />
              ) : null}

              <Table
                rowKey="serverId"
                columns={columns}
                dataSource={servers}
                loading={loading}
                pagination={{ pageSize: 8 }}
                locale={{
                  emptyText: (
                    <Empty
                      description="还没有 MCP 服务，先创建一个远程服务配置"
                      image={Empty.PRESENTED_IMAGE_SIMPLE}
                    />
                  )
                }}
              />
            </div>
          </div>
        </section>

        <Drawer
          title={editingServerId ? '编辑 MCP 服务' : '新增 MCP 服务'}
          width={720}
          open={drawerOpen}
          onClose={handleDrawerClose}
          destroyOnHidden
          loading={drawerLoading}
          extra={(
            <Space>
              <Button onClick={handleDrawerClose}>取消</Button>
              <Button onClick={handleDrawerTest} loading={testing} disabled={!editingServerId}>
                测试连接
              </Button>
              <Button type="primary" onClick={() => void handleSubmit()} loading={saving}>
                保存
              </Button>
            </Space>
          )}
        >
            <Form
              form={form}
              layout="vertical"
              initialValues={{ transportType: 'sse', headers: [] }}
            >
            <Form.Item
              label="服务名称"
              name="name"
              rules={[{ required: true, message: '请输入服务名称' }]}
            >
              <Input placeholder="例如：Docs Search MCP" />
            </Form.Item>

            <Form.Item
              label="传输方式"
              name="transportType"
              rules={[{ required: true, message: '请选择传输方式' }]}
            >
              <Select
                options={[
                  { label: 'SSE', value: 'sse' },
                  { label: 'HTTP', value: 'http' }
                ]}
              />
            </Form.Item>

            <Form.Item
              label="Endpoint"
              name="endpoint"
              rules={[{ required: true, message: '请输入 Endpoint' }]}
            >
              <Input placeholder="https://example.com/mcp" />
            </Form.Item>

            <Form.List name="headers">
              {(fields, { add, remove }) => (
                <div className="mcp-manager-drawer-section">
                  <Space style={{ marginBottom: 12 }}>
                    <strong>自定义请求头</strong>
                    <Button type="dashed" icon={<PlusOutlined />} onClick={() => add({ key: '', value: '', hasValue: false, maskedValue: '' })}>
                      添加 Header
                    </Button>
                  </Space>

                  {fields.length === 0 ? (
                    <div className="mcp-manager-empty-state">当前没有额外请求头</div>
                  ) : null}

                  {fields.map((field) => {
                    const maskedValue = form.getFieldValue(['headers', field.name, 'maskedValue'])
                    const hasValue = hasSavedHeaderValue(form.getFieldValue(['headers', field.name, 'hasValue']))

                    return (
                      <div key={field.key} className="mcp-manager-header-row">
                        <Form.Item
                          {...field}
                          label="Key"
                          name={[field.name, 'key']}
                          rules={[{ required: true, message: '请输入 Header Key' }]}
                        >
                          <Input placeholder="Authorization" />
                        </Form.Item>

                        <Form.Item
                          {...field}
                          label="Value"
                          name={[field.name, 'value']}
                          extra={hasValue ? '留空表示保留已保存的加密值' : null}
                        >
                          <Input.Password
                            placeholder={maskedValue || 'Bearer xxx'}
                            visibilityToggle={false}
                          />
                        </Form.Item>

                        <Form.Item label="操作">
                          <Button danger icon={<DeleteOutlined />} onClick={() => remove(field.name)}>
                            删除
                          </Button>
                        </Form.Item>

                        <Form.Item name={[field.name, 'hasValue']} hidden>
                          <Input type="hidden" />
                        </Form.Item>
                        <Form.Item name={[field.name, 'maskedValue']} hidden>
                          <Input type="hidden" />
                        </Form.Item>
                      </div>
                    )
                  })}
                </div>
              )}
            </Form.List>
          </Form>

          <div className="mcp-manager-drawer-section">
            <Space style={{ marginBottom: 12 }}>
              <ApiOutlined />
              <strong>最近一次工具快照</strong>
            </Space>

            {drawerTools.length === 0 ? (
              <div className="mcp-manager-empty-state">还没有工具快照。保存后点击“测试连接”即可刷新。</div>
            ) : (
              <div className="mcp-manager-tool-list">
                {drawerTools.map((tool) => (
                  <div key={tool.apiName} className="mcp-manager-tool-card">
                    <strong>{tool.displayName}</strong>
                    <span className="mcp-manager-hint">{tool.apiName}</span>
                    <p>{tool.description || '暂无描述'}</p>
                  </div>
                ))}
              </div>
            )}
          </div>
        </Drawer>

        <Modal
          title={previewServer ? `工具预览 · ${previewServer.name}` : '工具预览'}
          open={Boolean(previewServer)}
          onCancel={() => setPreviewServer(null)}
          footer={null}
          width={820}
        >
          {previewServer?.tools?.length ? (
            <div className="mcp-manager-tool-list">
              {previewServer.tools.map((tool) => (
                <div key={tool.apiName} className="mcp-manager-tool-card">
                  <strong>{tool.displayName}</strong>
                  <span className="mcp-manager-hint">{tool.apiName}</span>
                  <p>{tool.description || '暂无描述'}</p>
                </div>
              ))}
            </div>
          ) : (
            <Empty description="当前没有工具快照，请先测试连接" image={Empty.PRESENTED_IMAGE_SIMPLE} />
          )}
        </Modal>
      </Content>
    </Layout>
  )
}

export default MCPManager
