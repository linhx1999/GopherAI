import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Form, Input, Button, Card, App } from 'antd'
import { UserOutlined, LockOutlined } from '@ant-design/icons'
import api from '../utils/api'
import './Login.css'

const Login = () => {
  const navigate = useNavigate()
  const { message } = App.useApp()
  const [loading, setLoading] = useState(false)
  const [form] = Form.useForm()

  const handleLogin = async (values) => {
    try {
      setLoading(true)
      const response = await api.post('/user/login', {
        username: values.username,
        password: values.password
      })
      if (response.data.code === 1000) {
        localStorage.setItem('token', response.data.data[0].token)
        message.success('登录成功')
        navigate('/menu')
      } else {
        message.error(response.data.msg || '登录失败')
      }
    } catch (error) {
      console.error('Login error:', error)
      message.error('登录失败，请重试')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-container">
      <Card className="login-card" bordered={false}>
        <div className="card-header">
          <h2>登录</h2>
        </div>
        <Form
          form={form}
          onFinish={handleLogin}
          layout="vertical"
          autoComplete="off"
        >
          <Form.Item
            label="用户名"
            name="username"
            rules={[
              { required: true, message: '请输入用户名' }
            ]}
          >
            <Input
              prefix={<UserOutlined />}
              placeholder="请输入用户名"
              size="large"
            />
          </Form.Item>
          <Form.Item
            label="密码"
            name="password"
            rules={[
              { required: true, message: '请输入密码' },
              { min: 6, message: '密码长度不能少于6位' }
            ]}
          >
            <Input.Password
              prefix={<LockOutlined />}
              placeholder="请输入密码"
              size="large"
            />
          </Form.Item>
          <Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              loading={loading}
              block
              size="large"
            >
              登录
            </Button>
          </Form.Item>
          <Form.Item>
            <Button
              type="link"
              onClick={() => navigate('/register')}
              block
            >
              还没有账号？去注册
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  )
}

export default Login