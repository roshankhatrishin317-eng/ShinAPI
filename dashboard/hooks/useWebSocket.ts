'use client'

import { useEffect, useRef, useState, useCallback } from 'react'

type ConnectionState = 'connecting' | 'connected' | 'disconnected' | 'reconnecting'

interface UseWebSocketOptions {
  url: string
  authKey: string | null
  onMessage: (data: any) => void
  reconnectInterval?: number
  maxReconnectAttempts?: number
}

interface UseWebSocketResult {
  connectionState: ConnectionState
  reconnectAttempts: number
  lastMessageTime: number | null
  connect: () => void
  disconnect: () => void
}

export function useMetricsWebSocket({
  url,
  authKey,
  onMessage,
  reconnectInterval = 1000,
  maxReconnectAttempts = 10
}: UseWebSocketOptions): UseWebSocketResult {
  const [connectionState, setConnectionState] = useState<ConnectionState>('disconnected')
  const [reconnectAttempts, setReconnectAttempts] = useState(0)
  const [lastMessageTime, setLastMessageTime] = useState<number | null>(null)
  
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const shouldReconnectRef = useRef(true)
  const reconnectAttemptsRef = useRef(0)
  const onMessageRef = useRef(onMessage)

  // Keep onMessage ref updated to avoid stale closures
  useEffect(() => {
    onMessageRef.current = onMessage
  }, [onMessage])

  // Sync reconnectAttempts state with ref
  useEffect(() => {
    reconnectAttemptsRef.current = reconnectAttempts
  }, [reconnectAttempts])

  const connect = useCallback(() => {
    if (!authKey || !url) return
    if (wsRef.current?.readyState === WebSocket.OPEN) return
    if (wsRef.current?.readyState === WebSocket.CONNECTING) return

    shouldReconnectRef.current = true
    setConnectionState('connecting')

    try {
      const wsUrl = `${url}?key=${encodeURIComponent(authKey)}`
      const ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onopen = () => {
        setConnectionState('connected')
        setReconnectAttempts(0)
        reconnectAttemptsRef.current = 0
      }

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          setLastMessageTime(Date.now())
          onMessageRef.current(data)
        } catch (e) {
          console.error('Failed to parse WebSocket message:', e)
        }
      }

      ws.onerror = (error) => {
        console.error('WebSocket error:', error)
      }

      ws.onclose = () => {
        wsRef.current = null
        
        if (shouldReconnectRef.current && reconnectAttemptsRef.current < maxReconnectAttempts) {
          setConnectionState('reconnecting')
          const currentAttempts = reconnectAttemptsRef.current
          const backoff = Math.min(reconnectInterval * Math.pow(2, currentAttempts), 30000)
          
          reconnectTimeoutRef.current = setTimeout(() => {
            setReconnectAttempts(prev => prev + 1)
            reconnectAttemptsRef.current += 1
            connect()
          }, backoff)
        } else {
          setConnectionState('disconnected')
        }
      }
    } catch (error) {
      console.error('Failed to create WebSocket:', error)
      setConnectionState('disconnected')
    }
  }, [url, authKey, reconnectInterval, maxReconnectAttempts])

  const disconnect = useCallback(() => {
    shouldReconnectRef.current = false
    
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }
    
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }
    
    setConnectionState('disconnected')
    setReconnectAttempts(0)
    reconnectAttemptsRef.current = 0
  }, [])

  useEffect(() => {
    if (authKey && url) {
      connect()
    }
    
    return () => {
      disconnect()
    }
  }, [authKey, url, connect, disconnect])

  return {
    connectionState,
    reconnectAttempts,
    lastMessageTime,
    connect,
    disconnect
  }
}
