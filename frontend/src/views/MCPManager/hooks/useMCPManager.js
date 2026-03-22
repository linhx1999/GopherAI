import { useCallback, useEffect, useState } from 'react'
import api from '../../../utils/api'
import { API_ENDPOINTS, STATUS_CODES } from '../../Chat/config/constants'

const parseToolItem = (tool) => {
  const name = String(tool?.name || tool?.api_name || '').trim()
  if (!name) {
    return null
  }

  return {
    apiName: name,
    displayName: String(tool?.display_name || tool?.displayName || name).trim(),
    description: String(tool?.description || '').trim(),
  }
}

const parseServer = (server) => {
  const serverId = String(server?.server_id || server?.serverId || '').trim()
  if (!serverId) {
    return null
  }

  return {
    serverId,
    name: String(server?.name || '').trim(),
    transportType: String(server?.transport_type || server?.transportType || 'sse').trim().toLowerCase(),
    endpoint: String(server?.endpoint || '').trim(),
    lastTestStatus: String(server?.last_test_status || server?.lastTestStatus || 'untested').trim(),
    lastTestMessage: String(server?.last_test_message || server?.lastTestMessage || '').trim(),
    lastTestedAt: String(server?.last_tested_at || server?.lastTestedAt || '').trim(),
    createdAt: String(server?.created_at || server?.createdAt || '').trim(),
    headers: Array.isArray(server?.headers)
      ? server.headers.map((item) => ({
        key: String(item?.key || '').trim(),
        maskedValue: String(item?.masked_value || item?.maskedValue || '').trim(),
        hasValue: Boolean(item?.has_value ?? item?.hasValue)
      })).filter((item) => item.key)
      : [],
    tools: Array.isArray(server?.tools)
      ? server.tools.map(parseToolItem).filter(Boolean)
      : []
  }
}

const useMCPManager = () => {
  const [servers, setServers] = useState([])
  const [loading, setLoading] = useState(false)
  const [featureDisabled, setFeatureDisabled] = useState(false)
  const [errorMessage, setErrorMessage] = useState('')

  const loadServers = useCallback(async () => {
    setLoading(true)
    try {
      const response = await api.get(API_ENDPOINTS.MCP_SERVERS)
      if (response.data?.code === STATUS_CODES.SUCCESS) {
        const items = Array.isArray(response.data?.data?.servers)
          ? response.data.data.servers.map(parseServer).filter(Boolean)
          : []

        setServers(items)
        setFeatureDisabled(false)
        setErrorMessage('')
        return items
      }

      setServers([])
      setFeatureDisabled(String(response.data?.msg || '').includes('MCP 功能未启用'))
      setErrorMessage(response.data?.msg || '获取 MCP 服务失败')
      return []
    } catch (error) {
      console.error('Load MCP servers error:', error)
      setServers([])
      setFeatureDisabled(false)
      setErrorMessage('获取 MCP 服务失败')
      return []
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadServers()
  }, [loadServers])

  const fetchServerDetail = useCallback(async (serverId) => {
    const response = await api.get(API_ENDPOINTS.MCP_SERVER_DETAIL(serverId))
    if (response.data?.code !== STATUS_CODES.SUCCESS) {
      throw new Error(response.data?.msg || '获取 MCP 服务详情失败')
    }

    const server = parseServer(response.data?.data?.server)
    if (!server) {
      throw new Error('获取 MCP 服务详情失败')
    }

    return server
  }, [])

  const createServer = useCallback(async (payload) => {
    const response = await api.post(API_ENDPOINTS.MCP_SERVERS, payload)
    if (response.data?.code !== STATUS_CODES.SUCCESS) {
      throw new Error(response.data?.msg || '创建 MCP 服务失败')
    }
    return parseServer(response.data?.data?.server)
  }, [])

  const updateServer = useCallback(async (serverId, payload) => {
    const response = await api.put(API_ENDPOINTS.MCP_SERVER_DETAIL(serverId), payload)
    if (response.data?.code !== STATUS_CODES.SUCCESS) {
      throw new Error(response.data?.msg || '更新 MCP 服务失败')
    }
    return parseServer(response.data?.data?.server)
  }, [])

  const deleteServer = useCallback(async (serverId) => {
    const response = await api.delete(API_ENDPOINTS.MCP_SERVER_DETAIL(serverId))
    if (response.data?.code !== STATUS_CODES.SUCCESS) {
      throw new Error(response.data?.msg || '删除 MCP 服务失败')
    }
  }, [])

  const testServer = useCallback(async (serverId) => {
    const response = await api.post(API_ENDPOINTS.MCP_SERVER_TEST(serverId))
    const server = parseServer(response.data?.data?.server)
    if (response.data?.code !== STATUS_CODES.SUCCESS) {
      throw new Error(response.data?.msg || 'MCP 连接测试失败')
    }
    return server
  }, [])

  return {
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
  }
}

export default useMCPManager
