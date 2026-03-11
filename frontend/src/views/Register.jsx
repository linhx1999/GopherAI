import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Form, Input, Button, Card, App, Row, Col } from 'antd'
import { MailOutlined, LockOutlined, SafetyOutlined } from '@ant-design/icons'
import api from '../utils/api'
import './Register.css'

const Register = () => {
  const navigate = useNavigate()
  const { message } = App.useApp()
  const [loading, setLoading] = useState(false)
  const [codeLoading, setCodeLoading] = useState(false)
  const [countdown, setCountdown] = useState(0)
  const [form] = Form.useForm()

  const sendCode = async () => {
    const email = form.getFieldValue('email')
    if (!email) {
      message.warning('请先输入邮箱')
      return
    }
    try {
      setCodeLoading(true)
      const response = await api.post('/user/captcha', { email })
      if (response.data.code === 1000) {
        message.success('验证码发送成功')
        setCountdown(60)
        const timer = setInterval(() => {
          setCountdown((prev) => {
            if (prev <= 1) {
              clearInterval(timer)
              return 0
            }
            return prev - 1
          })
        }, 1000)
      } else {
        message.error(response.data.msg || '验证码发送失败')
      }
    } catch (error) {
      console.error('Send code error:', error)
      message.error('验证码发送失败，请重试')
    } finally {
      setCodeLoading(false)
    }
  }

  const handleRegister = async (values) => {
    try {
      setLoading(true)
      const response = await api.post('/user/register', {
        email: values.email,
        captcha: values.captcha,
        password: values.password
      })
      if (response.data.code === 1000) {
        message.success('注册成功，请登录')
        navigate('/login')
      } else {
        message.error(response.data.msg || '注册失败')
      }
    } catch (error) {
      console.error('Register error:', error)
      message.error('注册失败，请重试')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="register-container">
      <Card className="register-card" variant="borderless">
        <div className="card-header">
          <h2>注册</h2>
        </div>
        <Form
          form={form}
          onFinish={handleRegister}
          layout="vertical"
          autoComplete="off"
        >
          <Form.Item
            label="邮箱"
            name="email"
            rules={[
              { required: true, message: '请输入邮箱' },
              { type: 'email', message: '请输入正确的邮箱格式' }
            ]}
          >
            <Input
              prefix={<MailOutlined />}
              placeholder="请输入邮箱"
              size="large"
            />
          </Form.Item>
          <Form.Item
            label="验证码"
            name="captcha"
            rules={[
              { required: true, message: '请输入验证码' }
            ]}
          >
            <Row gutter={10}>
              <Col span={16}>
                <Input
                  prefix={<SafetyOutlined />}
                  placeholder="请输入验证码"
                  size="large"
                />
              </Col>
              <Col span={8}>
                <Button
                  type="primary"
                  loading={codeLoading}
                  disabled={countdown > 0}
                  onClick={sendCode}
                  block
                  size="large"
                >
                  {countdown > 0 ? `${countdown}s` : '发送验证码'}
                </Button>
              </Col>
            </Row>
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
          <Form.Item
            label="确认密码"
            name="confirmPassword"
            dependencies={['password']}
            rules={[
              { required: true, message: '请确认密码' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('password') === value) {
                    return Promise.resolve()
                  }
                  return Promise.reject(new Error('两次输入密码不一致'))
                }
              })
            ]}
          >
            <Input.Password
              prefix={<LockOutlined />}
              placeholder="请再次输入密码"
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
              注册
            </Button>
          </Form.Item>
          <Form.Item>
            <Button
              type="link"
              onClick={() => navigate('/login')}
              block
            >
              已有账号？去登录
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  )
}

export default Register
