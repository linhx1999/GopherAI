import { useEffect, useState } from 'react'
import { App, Layout, Table, Button, Space, Upload, Modal, Tag, Tooltip, Popconfirm } from 'antd'
import { UploadOutlined, CloudUploadOutlined, FileTextOutlined, DeleteOutlined, ReloadOutlined, CheckCircleOutlined, LoadingOutlined, CloseCircleOutlined } from '@ant-design/icons'
import api from '../../utils/api'
import { API_ENDPOINTS, STATUS_CODES } from '../Chat/config/constants'
import './index.css'

const { Header, Content } = Layout

const FileManager = () => {
  const { message } = App.useApp()
  const [files, setFiles] = useState([])
  const [loading, setLoading] = useState(false)
  const [uploading, setUploading] = useState(false)

  // 获取文件列表
  const fetchFiles = async () => {
    setLoading(true)
    try {
      const response = await api.get(API_ENDPOINTS.FILE_LIST)
      if (response.data.code === STATUS_CODES.SUCCESS) {
        setFiles(response.data.data || [])
      } else {
        message.error(response.data.msg || '获取文件列表失败')
      }
    } catch (error) {
      message.error('获取文件列表失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    let mounted = true
    fetchFiles()
    return () => {
      mounted = false
    }
  }, [])

  // 上传文件
  const handleUpload = async (file) => {
    const formData = new FormData()
    formData.append('file', file)
    setUploading(true)

    try {
      const response = await api.post(API_ENDPOINTS.FILE_UPLOAD, formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        }
      })
      if (response.data.code === STATUS_CODES.SUCCESS) {
        message.success('文件上传成功')
        fetchFiles()
      } else {
        message.error(response.data.msg || '文件上传失败')
      }
    } catch (error) {
      message.error('文件上传失败')
    } finally {
      setUploading(false)
    }

    return false
  }

  // 触发索引
  const handleIndex = async (fileId) => {
    try {
      const response = await api.post(API_ENDPOINTS.FILE_INDEX(fileId))
      if (response.data.code === STATUS_CODES.SUCCESS) {
        message.success('索引任务已创建，请稍后查看状态')
        fetchFiles()
      } else {
        message.error(response.data.msg || '创建索引失败')
      }
    } catch (error) {
      message.error('创建索引失败')
    }
  }

  // 删除索引
  const handleDeleteIndex = async (fileId) => {
    try {
      const response = await api.delete(API_ENDPOINTS.FILE_INDEX_DELETE(fileId))
      if (response.data.code === STATUS_CODES.SUCCESS) {
        message.success('索引删除成功')
        fetchFiles()
      } else {
        message.error(response.data.msg || '删除索引失败')
      }
    } catch (error) {
      message.error('删除索引失败')
    }
  }

  // 删除文件
  const handleDeleteFile = async (fileId) => {
    try {
      const response = await api.delete(API_ENDPOINTS.FILE_DELETE(fileId))
      if (response.data.code === STATUS_CODES.SUCCESS) {
        message.success('文件删除成功')
        fetchFiles()
      } else {
        message.error(response.data.msg || '删除文件失败')
      }
    } catch (error) {
      message.error('删除文件失败')
    }
  }

  // 获取索引状态标签
  const getIndexStatusTag = (status) => {
    switch (status) {
      case 'pending':
        return <Tag icon={<ReloadOutlined />} color="default">未索引</Tag>
      case 'indexing':
        return <Tag icon={<LoadingOutlined />} color="processing">索引中</Tag>
      case 'indexed':
        return <Tag icon={<CheckCircleOutlined />} color="success">已索引</Tag>
      case 'failed':
        return <Tag icon={<CloseCircleOutlined />} color="error">索引失败</Tag>
      default:
        return <Tag>未知</Tag>
    }
  }

  const columns = [
    {
      title: '文件名',
      dataIndex: 'file_name',
      key: 'file_name',
      render: (text, record) => (
        <Space>
          <FileTextOutlined />
          {text}
        </Space>
      )
    },
    {
      title: '文件大小',
      dataIndex: 'file_size',
      key: 'file_size'
    },
    {
      title: '文件类型',
      dataIndex: 'file_type',
      key: 'file_type',
      render: (text) => <Tag>{text}</Tag>
    },
    {
      title: '索引状态',
      dataIndex: 'index_status',
      key: 'index_status',
      render: (status, record) => (
        <Space direction="vertical" size={0}>
          {getIndexStatusTag(status)}
          {record.index_message && (
            <Tooltip title={record.index_message}>
              <span className="index-message">{record.index_message}</span>
            </Tooltip>
          )}
        </Space>
      )
    },
    {
      title: '上传时间',
      dataIndex: 'created_at',
      key: 'created_at'
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space size="small">
          {record.index_status === 'pending' || record.index_status === 'failed' ? (
            <Tooltip title="创建索引">
              <Button
                type="primary"
                size="small"
                icon={<CloudUploadOutlined />}
                onClick={() => handleIndex(record.id)}
              >
                索引
              </Button>
            </Tooltip>
          ) : null}
          {record.index_status === 'indexed' ? (
            <Tooltip title="删除索引">
              <Button
                size="small"
                icon={<ReloadOutlined />}
                onClick={() => handleDeleteIndex(record.id)}
              >
                删除索引
              </Button>
            </Tooltip>
          ) : null}
          <Popconfirm
            title="确定要删除这个文件吗？"
            description="删除后将无法恢复，同时会删除相关索引"
            onConfirm={() => handleDeleteFile(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Tooltip title="删除文件">
              <Button
                danger
                size="small"
                icon={<DeleteOutlined />}
              >
                删除
              </Button>
            </Tooltip>
          </Popconfirm>
        </Space>
      )
    }
  ]

  return (
    <Layout className="file-manager-container">
      <Header className="file-manager-header">
        <h1>文件管理</h1>
        <Space>
          <Upload
            accept=".md,.txt"
            showUploadList={false}
            beforeUpload={handleUpload}
            disabled={uploading}
          >
            <Button
              type="primary"
              icon={<UploadOutlined />}
              loading={uploading}
            >
              上传文件
            </Button>
          </Upload>
          <Button
            icon={<ReloadOutlined />}
            onClick={fetchFiles}
            loading={loading}
          >
            刷新
          </Button>
        </Space>
      </Header>
      <Content className="file-manager-content">
        <div className="table-container">
          <Table
            columns={columns}
            dataSource={files}
            rowKey="id"
            loading={loading}
            pagination={{
              pageSize: 10,
              showSizeChanger: true,
              showTotal: (total) => `共 ${total} 个文件`
            }}
          />
        </div>
      </Content>
    </Layout>
  )
}

export default FileManager
