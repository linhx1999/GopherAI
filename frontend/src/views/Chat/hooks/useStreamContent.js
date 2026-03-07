import { useState, useRef, useCallback, useEffect } from 'react'

/**
 * 流式内容 Hook - 参考 Ant Design X 官方示例
 * 用于实现逐字显示的打字效果
 * @param {string} content - 要显示的内容
 * @param {object} options - 配置选项
 * @param {number} options.step - 每次显示的字符数，默认 2
 * @param {number} options.interval - 每次显示的间隔时间(ms)，默认 50
 * @returns {[string, boolean]} - [当前显示的内容, 是否完成]
 */
function useStreamContent(
  content,
  { step = 2, interval = 50 } = {}
) {
  const [streamContent, setStreamContent] = useState('')
  const streamRef = useRef('')
  const doneRef = useRef(true)
  const timerRef = useRef(-1)
  const stepRef = useRef(step)
  const intervalRef = useRef(interval)

  // 使用 useEffect 更新 ref
  useEffect(() => {
    stepRef.current = step
    intervalRef.current = interval
  }, [step, interval])

  // 流式开始函数
  const startStream = useCallback((text) => {
    doneRef.current = false
    streamRef.current = ''
    timerRef.current = setInterval(() => {
      const len = streamRef.current.length + stepRef.current
      if (len <= text.length - 1) {
        const newContent = text.slice(0, len)
        setStreamContent(newContent)
        streamRef.current = newContent
      } else {
        setStreamContent(text)
        streamRef.current = text
        doneRef.current = true
        clearInterval(timerRef.current)
      }
    }, intervalRef.current)
  }, [])

  useEffect(() => {
    // 内容相同，不处理
    if (content === streamRef.current) return

    // 清空内容
    if (!content && streamRef.current) {
      setStreamContent('')
      doneRef.current = true
      clearInterval(timerRef.current)
      return
    }

    // 新内容开始流式
    if (!streamRef.current && content) {
      clearInterval(timerRef.current)
      startStream(content)
    } else if (content.indexOf(streamRef.current) !== 0) {
      // 非起始子集认为是全新内容，重新开始流式
      clearInterval(timerRef.current)
      startStream(content)
    }
  }, [content, startStream])

  // 清理定时器
  useEffect(() => {
    return () => clearInterval(timerRef.current)
  }, [])

  // 使用 state 来跟踪 done 状态
  const [isDone, setIsDone] = useState(true)
  useEffect(() => {
    const checkDone = setInterval(() => {
      setIsDone(doneRef.current)
    }, 50)
    return () => clearInterval(checkDone)
  }, [])

  return [streamContent, isDone]
}

export default useStreamContent
