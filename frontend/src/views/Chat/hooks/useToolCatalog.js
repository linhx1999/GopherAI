import { useCallback, useEffect, useState } from 'react'
import api from '../../../utils/api'
import { API_ENDPOINTS, STATUS_CODES } from '../config/constants'

const parseToolCatalogResponse = (responseData) => {
  const toolList = responseData?.data?.tools
  if (!Array.isArray(toolList)) {
    return []
  }

  return toolList
    .map((tool) => ({
      apiName: String(tool?.name || tool?.api_name || '').trim(),
      displayName: String(tool?.display_name || tool?.displayName || tool?.name || tool?.api_name || '').trim(),
      description: String(tool?.description || '').trim()
    }))
    .filter((tool) => tool.apiName)
}

const parseMCPServerToolItem = (tool) => {
  const apiName = String(tool?.name || tool?.api_name || '').trim()
  if (!apiName) {
    return null
  }

  return {
    apiName,
    displayName: String(tool?.display_name || tool?.displayName || apiName).trim(),
    description: String(tool?.description || '').trim()
  }
}

const parseMCPServerCatalogResponse = (responseData) => {
  const serverList = responseData?.data?.mcp_servers
  if (!Array.isArray(serverList)) {
    return []
  }

  return serverList
    .map((server) => {
      const serverId = String(server?.server_id || server?.serverId || '').trim()
      if (!serverId) {
        return null
      }

      return {
        serverId,
        name: String(server?.name || '').trim(),
        transportType: String(server?.transport_type || server?.transportType || 'sse').trim(),
        endpoint: String(server?.endpoint || '').trim(),
        lastTestStatus: String(server?.last_test_status || server?.lastTestStatus || 'untested').trim(),
        lastTestMessage: String(server?.last_test_message || server?.lastTestMessage || '').trim(),
        lastTestedAt: String(server?.last_tested_at || server?.lastTestedAt || '').trim(),
        tools: Array.isArray(server?.tools)
          ? server.tools.map(parseMCPServerToolItem).filter(Boolean)
          : []
      }
    })
    .filter(Boolean)
}

const useToolCatalog = () => {
  const [availableTools, setAvailableTools] = useState([])
  const [availableMCPServers, setAvailableMCPServers] = useState([])
  const [mcpFeatureEnabled, setMCPFeatureEnabled] = useState(false)

  const loadToolCatalog = useCallback(async () => {
    try {
      const response = await api.get(API_ENDPOINTS.TOOLS)
      if (response.data?.code === STATUS_CODES.SUCCESS) {
        setAvailableTools(parseToolCatalogResponse(response.data))
        setAvailableMCPServers(parseMCPServerCatalogResponse(response.data))
        setMCPFeatureEnabled(Boolean(response.data?.data?.mcp_feature_enabled))
        return
      }

      setAvailableTools([])
      setAvailableMCPServers([])
      setMCPFeatureEnabled(false)
    } catch (error) {
      console.error('Load tool catalog error:', error)
      setAvailableTools([])
      setAvailableMCPServers([])
      setMCPFeatureEnabled(false)
    }
  }, [])

  useEffect(() => {
    queueMicrotask(() => {
      void loadToolCatalog()
    })
  }, [loadToolCatalog])

  return {
    availableTools,
    availableMCPServers,
    mcpFeatureEnabled,
    reloadToolCatalog: loadToolCatalog
  }
}

export default useToolCatalog
