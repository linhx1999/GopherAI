import axios from 'axios'
import { getStoredToken, handleUnauthorized, isUnauthorizedResponseCode } from './auth'

export const API_BASE_URL = '/api/v1'

const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 0
})

// 请求拦截器
api.interceptors.request.use(
  (config) => {
    const token = getStoredToken()
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
api.interceptors.response.use(
  (response) => {
    if (isUnauthorizedResponseCode(response.data?.code)) {
      handleUnauthorized()
      return Promise.reject(new Error(response.data?.msg || '登录已失效'))
    }
    return response
  },
  (error) => {
    if (error.response && error.response.status === 401) {
      handleUnauthorized()
    }
    return Promise.reject(error)
  }
)

export default api
