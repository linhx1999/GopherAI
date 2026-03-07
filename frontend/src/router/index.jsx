import { createBrowserRouter, Navigate } from 'react-router-dom'
import Login from '../views/Login'
import Register from '../views/Register'
import Menu from '../views/Menu'
import AIChat from '../views/AIChat'
import ImageRecognition from '../views/ImageRecognition'
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
        path: '/ai-chat',
        element: <AIChat />
      },
      {
        path: '/image-recognition',
        element: <ImageRecognition />
      },
      {
        path: '/file-manager',
        element: <FileManager />
      }
    ]
  }
])

export default router
