import { Button, Space, Upload } from 'antd'
import { ReloadOutlined, UploadOutlined } from '@ant-design/icons'
import { SUPPORTED_FILE_TYPES } from '../../Chat/config/constants'

const FileManagerToolbar = ({
  fileSummary,
  loading,
  uploading,
  onRefresh,
  onUpload
}) => {
  const acceptedTypes = SUPPORTED_FILE_TYPES.join(',')

  return (
    <div className="file-manager-panel-toolbar">
      <div className="file-manager-panel-copy">
        <span className="file-manager-panel-eyebrow">Workspace</span>
        <h2>知识库文件</h2>
        <p>上传 Markdown 或文本文件，并按需创建或删除索引。</p>
      </div>

      <div className="file-manager-panel-actions">
        <div className="file-manager-stats">
          <div className="file-manager-stat-card">
            <span className="file-manager-stat-label">文件总数</span>
            <strong>{fileSummary.total}</strong>
          </div>
          <div className="file-manager-stat-card">
            <span className="file-manager-stat-label">已索引</span>
            <strong>{fileSummary.indexed}</strong>
          </div>
          <div className="file-manager-stat-card">
            <span className="file-manager-stat-label">索引中</span>
            <strong>{fileSummary.indexing}</strong>
          </div>
          <div className="file-manager-stat-card">
            <span className="file-manager-stat-label">失败</span>
            <strong>{fileSummary.failed}</strong>
          </div>
        </div>

        <Space wrap>
          <Upload
            accept={acceptedTypes}
            showUploadList={false}
            beforeUpload={onUpload}
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
            onClick={onRefresh}
            loading={loading}
          >
            刷新列表
          </Button>
        </Space>
      </div>
    </div>
  )
}

export default FileManagerToolbar
