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

const useToolCatalog = () => {
  const [availableTools, setAvailableTools] = useState([])

  const loadToolCatalog = useCallback(async () => {
    try {
      const response = await api.get(API_ENDPOINTS.TOOLS)
      if (response.data?.code === STATUS_CODES.SUCCESS) {
        setAvailableTools(parseToolCatalogResponse(response.data))
      }
    } catch (error) {
      console.error('Load tool catalog error:', error)
      setAvailableTools([])
    }
  }, [])

  useEffect(() => {
    queueMicrotask(() => {
      void loadToolCatalog()
    })
  }, [loadToolCatalog])

  return {
    availableTools,
    reloadToolCatalog: loadToolCatalog
  }
}

export default useToolCatalog
