import { useState, useRef, useCallback, useEffect } from 'react'

const scheduleStateUpdate = (updater) => {
  queueMicrotask(updater)
}

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
  const [isDone, setIsDone] = useState(true)
  const streamRef = useRef('')
  const targetRef = useRef('')
  const timerRef = useRef(-1)
  const stepRef = useRef(step)
  const intervalRef = useRef(interval)

  // 使用 useEffect 更新 ref
  useEffect(() => {
    stepRef.current = step
    intervalRef.current = interval
  }, [step, interval])

  const stopStream = useCallback(() => {
    if (timerRef.current !== -1) {
      clearInterval(timerRef.current)
      timerRef.current = -1
    }
  }, [])

  // 流式开始函数
  const startStream = useCallback(() => {
    if (timerRef.current !== -1) return
    timerRef.current = setInterval(() => {
      const text = targetRef.current
      const len = streamRef.current.length + stepRef.current

      if (!text) {
        setStreamContent('')
        streamRef.current = ''
        scheduleStateUpdate(() => setIsDone(true))
        stopStream()
        return
      }

      if (len < text.length) {
        const newContent = text.slice(0, len)
        setStreamContent(newContent)
        streamRef.current = newContent
      } else {
        setStreamContent(text)
        streamRef.current = text
        scheduleStateUpdate(() => setIsDone(true))
        stopStream()
      }
    }, intervalRef.current)
  }, [stopStream])

  useEffect(() => {
    targetRef.current = content || ''

    // 清空内容
    if (!content && streamRef.current) {
      queueMicrotask(() => setStreamContent(''))
      streamRef.current = ''
      scheduleStateUpdate(() => setIsDone(true))
      stopStream()
      return
    }

    if (!content) {
      scheduleStateUpdate(() => setIsDone(true))
      return
    }

    // 非起始子集认为是全新内容，重新开始流式
    if (!content.startsWith(streamRef.current)) {
      stopStream()
      streamRef.current = ''
      scheduleStateUpdate(() => setIsDone(false))
      queueMicrotask(() => setStreamContent(''))
    }

    if (content !== streamRef.current) {
      scheduleStateUpdate(() => setIsDone(false))
      queueMicrotask(startStream)
    }
  }, [content, startStream, stopStream])

  // 清理定时器
  useEffect(() => {
    return () => stopStream()
  }, [stopStream])

  return [streamContent, isDone]
}

export default useStreamContent
