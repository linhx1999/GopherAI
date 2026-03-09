import { createBrowserRouter, Navigate } from 'react-router-dom'
import Login from '../views/Login'
import Register from '../views/Register'
import Menu from '../views/Menu'
import Chat from '../views/Chat'
import FileManager from '../views/FileManager'
import PrivateRoute from '../components/PrivateRoute'
import PublicRoute from '../components/PublicRoute'

const router = createBrowserRouter([
  {
    path: '/',
    element: <Navigate to="/login" replace />
  },
  {
    element: <PublicRoute />,
    children: [
      {
        path: '/login',
        element: <Login />
      },
      {
        path: '/register',
        element: <Register />
      }
    ]
  },
  {
    element: <PrivateRoute />,
    children: [
      {
        path: '/menu',
        element: <Menu />
      },
      {
        path: '/chat',
        element: <Chat />
      },
      {
        path: '/file-manager',
        element: <FileManager />
      }
    ]
  }
])

export default router
