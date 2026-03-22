let redirectingToLogin = false
const UNAUTHORIZED_RESPONSE_CODES = new Set([2006, 2007])

export const getStoredToken = () => localStorage.getItem('token')

export const clearStoredToken = () => {
  localStorage.removeItem('token')
}

export const isUnauthorizedResponseCode = (code) => UNAUTHORIZED_RESPONSE_CODES.has(Number(code))

export const handleUnauthorized = () => {
  clearStoredToken()

  if (redirectingToLogin) {
    return
  }

  if (window.location.pathname === '/login') {
    return
  }

  redirectingToLogin = true
  window.location.replace('/login')
}
