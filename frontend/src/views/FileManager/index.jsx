import { Layout, theme } from 'antd'
import FileManagerHeader from './components/FileManagerHeader'
import FileManagerToolbar from './components/FileManagerToolbar'
import FileTable from './components/FileTable'
import useFileManager from './hooks/useFileManager'
import './index.css'

const { Content } = Layout

const FileManager = () => {
  const { token } = theme.useToken()
  const pageStyle = {
    minHeight: '100vh',
    background: token.colorBgLayout,
  }
  const {
    files,
    loading,
    uploading,
    fileSummary,
    loadFiles,
    uploadFile,
    createFileIndex,
    removeFileIndex,
    removeFile
  } = useFileManager()

  return (
    <Layout style={pageStyle} className="file-manager-layout">
      <Content className="file-manager-content">
        <FileManagerHeader />

        <section className="file-manager-shell">
          <div className="file-manager-panel">
            <FileManagerToolbar
              fileSummary={fileSummary}
              loading={loading}
              uploading={uploading}
              onRefresh={loadFiles}
              onUpload={uploadFile}
            />

            <div className="file-manager-table-panel">
              <FileTable
                files={files}
                loading={loading}
                onCreateIndex={createFileIndex}
                onRemoveIndex={removeFileIndex}
                onRemoveFile={removeFile}
              />
            </div>
          </div>
        </section>
      </Content>
    </Layout>
  )
}

export default FileManager
