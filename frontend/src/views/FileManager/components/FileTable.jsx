import { Button, Empty, Popconfirm, Space, Table, Tag, Tooltip } from 'antd'
import {
  CheckCircleOutlined,
  CloudUploadOutlined,
  CloseCircleOutlined,
  DeleteOutlined,
  FileTextOutlined,
  LoadingOutlined,
  ReloadOutlined
} from '@ant-design/icons'

const INDEX_STATUS_META = {
  pending: {
    color: 'default',
    icon: <ReloadOutlined />,
    label: '未索引'
  },
  indexing: {
    color: 'processing',
    icon: <LoadingOutlined />,
    label: '索引中'
  },
  indexed: {
    color: 'success',
    icon: <CheckCircleOutlined />,
    label: '已索引'
  },
  failed: {
    color: 'error',
    icon: <CloseCircleOutlined />,
    label: '索引失败'
  }
}

const renderIndexStatus = (status) => {
  const meta = INDEX_STATUS_META[status]

  if (!meta) {
    return <Tag>未知</Tag>
  }

  return (
    <Tag icon={meta.icon} color={meta.color}>
      {meta.label}
    </Tag>
  )
}

const FileTable = ({
  files,
  loading,
  onCreateIndex,
  onRemoveIndex,
  onRemoveFile
}) => {
  const columns = [
    {
      title: '文件名',
      dataIndex: 'file_name',
      key: 'file_name',
      render: (text) => (
        <Space>
          <FileTextOutlined />
          <span className="file-manager-file-name">{text}</span>
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
        <Space direction="vertical" size={4}>
          {renderIndexStatus(status)}
          {record.index_message ? (
            <Tooltip title={record.index_message}>
              <span className="file-manager-index-message">{record.index_message}</span>
            </Tooltip>
          ) : null}
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
        <Space size="small" wrap>
          {record.index_status === 'pending' || record.index_status === 'failed' ? (
            <Tooltip title="创建索引">
              <Button
                type="primary"
                size="small"
                icon={<CloudUploadOutlined />}
                onClick={() => onCreateIndex(record.id)}
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
                onClick={() => onRemoveIndex(record.id)}
              >
                删除索引
              </Button>
            </Tooltip>
          ) : null}

          <Popconfirm
            title="确定要删除这个文件吗？"
            description="删除后将无法恢复，同时会删除相关索引。"
            onConfirm={() => onRemoveFile(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Tooltip title="删除文件">
              <Button danger size="small" icon={<DeleteOutlined />}>
                删除
              </Button>
            </Tooltip>
          </Popconfirm>
        </Space>
      )
    }
  ]

  return (
    <Table
      columns={columns}
      dataSource={files}
      rowKey="id"
      loading={loading}
      scroll={{ x: 920 }}
      locale={{
        emptyText: (
          <Empty
            description="还没有文件，先上传一个 Markdown 或文本文件。"
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          />
        )
      }}
      pagination={{
        pageSize: 10,
        showSizeChanger: true,
        showTotal: (total) => `共 ${total} 个文件`
      }}
    />
  )
}

export default FileTable
