import { useCallback, useEffect, useMemo, useState } from 'react'
import { App } from 'antd'
import api from '../../../utils/api'
import { API_ENDPOINTS, STATUS_CODES } from '../../Chat/config/constants'

const useFileManager = () => {
  const { message } = App.useApp()
  const [files, setFiles] = useState([])
  const [loading, setLoading] = useState(false)
  const [uploading, setUploading] = useState(false)

  const loadFiles = useCallback(async () => {
    setLoading(true)

    try {
      const response = await api.get(API_ENDPOINTS.FILE_LIST)

      if (response.data.code === STATUS_CODES.SUCCESS) {
        setFiles(response.data.data || [])
        return
      }

      message.error(response.data.msg || '获取文件列表失败')
    } catch {
      message.error('获取文件列表失败')
    } finally {
      setLoading(false)
    }
  }, [message])

  useEffect(() => {
    void loadFiles()
  }, [loadFiles])

  const uploadFile = useCallback(async (file) => {
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
        await loadFiles()
      } else {
        message.error(response.data.msg || '文件上传失败')
      }
    } catch {
      message.error('文件上传失败')
    } finally {
      setUploading(false)
    }

    return false
  }, [loadFiles, message])

  const createFileIndex = useCallback(async (fileId) => {
    try {
      const response = await api.post(API_ENDPOINTS.FILE_INDEX(fileId))

      if (response.data.code === STATUS_CODES.SUCCESS) {
        message.success('索引任务已创建，请稍后查看状态')
        await loadFiles()
        return
      }

      message.error(response.data.msg || '创建索引失败')
    } catch {
      message.error('创建索引失败')
    }
  }, [loadFiles, message])

  const removeFileIndex = useCallback(async (fileId) => {
    try {
      const response = await api.delete(API_ENDPOINTS.FILE_INDEX_DELETE(fileId))

      if (response.data.code === STATUS_CODES.SUCCESS) {
        message.success('索引删除成功')
        await loadFiles()
        return
      }

      message.error(response.data.msg || '删除索引失败')
    } catch {
      message.error('删除索引失败')
    }
  }, [loadFiles, message])

  const removeFile = useCallback(async (fileId) => {
    try {
      const response = await api.delete(API_ENDPOINTS.FILE_DELETE(fileId))

      if (response.data.code === STATUS_CODES.SUCCESS) {
        message.success('文件删除成功')
        await loadFiles()
        return
      }

      message.error(response.data.msg || '删除文件失败')
    } catch {
      message.error('删除文件失败')
    }
  }, [loadFiles, message])

  const fileSummary = useMemo(() => ({
    total: files.length,
    indexed: files.filter(file => file.index_status === 'indexed').length,
    indexing: files.filter(file => file.index_status === 'indexing').length,
    failed: files.filter(file => file.index_status === 'failed').length,
  }), [files])

  return {
    files,
    loading,
    uploading,
    fileSummary,
    loadFiles,
    uploadFile,
    createFileIndex,
    removeFileIndex,
    removeFile
  }
}

export default useFileManager
