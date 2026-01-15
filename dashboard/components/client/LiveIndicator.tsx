'use client'

// Lightweight client component - Only handles WebSocket live updates
// Minimal JavaScript - no heavy libraries

import { useEffect, useState, useCallback } from 'react'
import { cn } from '@/lib/utils'

interface LiveMetrics {
  rpm: number
  tpm: number
  tps: number
  total_requests: number
  total_tokens: number
  total_success: number
  total_failed: number
  success_rate: number
  avg_latency_ms: number
}

interface LiveIndicatorProps {
  className?: string
}

export function LiveIndicator({ className }: LiveIndicatorProps) {
  const [isConnected, setIsConnected] = useState(false)
  const [metrics, setMetrics] = useState<LiveMetrics | null>(null)
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null)

  useEffect(() => {
    const key = localStorage.getItem('shinapi_mgmt_key')
    if (!key) return

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/ws/metrics?key=${encodeURIComponent(key)}`
    
    let ws: WebSocket | null = null
    let reconnectTimeout: NodeJS.Timeout

    const connect = () => {
      ws = new WebSocket(wsUrl)
      
      ws.onopen = () => {
        setIsConnected(true)
      }
      
      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          setMetrics(data)
          setLastUpdate(new Date())
        } catch (e) {
          console.error('Failed to parse metrics:', e)
        }
      }
      
      ws.onclose = () => {
        setIsConnected(false)
        // Reconnect after 3 seconds
        reconnectTimeout = setTimeout(connect, 3000)
      }
      
      ws.onerror = () => {
        setIsConnected(false)
      }
    }

    connect()

    return () => {
      clearTimeout(reconnectTimeout)
      ws?.close()
    }
  }, [])

  return (
    <div className={cn('flex items-center gap-4', className)}>
      {/* Connection status */}
      <div className="flex items-center gap-2">
        <span 
          className={cn(
            'w-2 h-2 rounded-full',
            isConnected ? 'bg-emerald-400 animate-pulse-dot' : 'bg-zinc-600'
          )}
        />
        <span className="text-xs text-zinc-400">
          {isConnected ? 'Live' : 'Disconnected'}
        </span>
      </div>

      {/* Live metrics */}
      {metrics && (
        <div className="flex items-center gap-4 text-sm">
          <div className="flex items-center gap-1">
            <span className="text-zinc-500">RPM:</span>
            <span className="text-white font-mono animate-number-tick" key={metrics.rpm}>
              {metrics.rpm}
            </span>
          </div>
          <div className="flex items-center gap-1">
            <span className="text-zinc-500">TPS:</span>
            <span className="text-cyan-400 font-mono animate-number-tick" key={metrics.tps}>
              {metrics.tps?.toFixed(1)}
            </span>
          </div>
          <div className="flex items-center gap-1">
            <span className="text-zinc-500">TPM:</span>
            <span className="text-violet-400 font-mono animate-number-tick" key={metrics.tpm}>
              {metrics.tpm >= 1000 ? `${(metrics.tpm / 1000).toFixed(1)}K` : metrics.tpm}
            </span>
          </div>
        </div>
      )}

      {/* Last update time */}
      {lastUpdate && (
        <span className="text-xs text-zinc-600 ml-auto">
          Updated {lastUpdate.toLocaleTimeString()}
        </span>
      )}
    </div>
  )
}

// Minimal live metrics display that updates via WebSocket
export function LiveMetricsBar() {
  const [metrics, setMetrics] = useState<LiveMetrics | null>(null)
  const [isConnected, setIsConnected] = useState(false)

  useEffect(() => {
    const key = localStorage.getItem('shinapi_mgmt_key')
    if (!key) return

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/ws/metrics?key=${encodeURIComponent(key)}`
    
    const ws = new WebSocket(wsUrl)
    
    ws.onopen = () => setIsConnected(true)
    ws.onclose = () => setIsConnected(false)
    ws.onmessage = (e) => {
      try {
        setMetrics(JSON.parse(e.data))
      } catch {}
    }

    return () => ws.close()
  }, [])

  if (!metrics) return null

  return (
    <div className="fixed bottom-0 left-0 right-0 bg-zinc-950/90 border-t border-zinc-800 backdrop-blur-sm">
      <div className="max-w-7xl mx-auto px-4 py-2 flex items-center gap-6">
        <div className="flex items-center gap-2">
          <span className={cn(
            'w-2 h-2 rounded-full',
            isConnected ? 'bg-emerald-400 animate-pulse-dot' : 'bg-red-400'
          )} />
          <span className="text-xs text-zinc-400">
            {isConnected ? 'Connected' : 'Reconnecting...'}
          </span>
        </div>

        <div className="flex items-center gap-6 text-sm">
          <Stat label="RPM" value={metrics.rpm} />
          <Stat label="TPM" value={metrics.tpm} color="text-violet-400" />
          <Stat label="TPS" value={metrics.tps} decimals={1} color="text-cyan-400" />
          <Stat label="Latency" value={metrics.avg_latency_ms} suffix="ms" color="text-amber-400" />
          <Stat label="Success" value={metrics.success_rate} suffix="%" color="text-emerald-400" decimals={1} />
        </div>
      </div>
    </div>
  )
}

function Stat({ 
  label, 
  value, 
  suffix, 
  color = 'text-white',
  decimals = 0
}: { 
  label: string
  value: number
  suffix?: string
  color?: string
  decimals?: number
}) {
  const formatted = value >= 1000000 
    ? `${(value / 1000000).toFixed(1)}M`
    : value >= 1000 
      ? `${(value / 1000).toFixed(1)}K`
      : decimals > 0 
        ? value.toFixed(decimals)
        : Math.round(value).toString()

  return (
    <div className="flex items-center gap-1">
      <span className="text-zinc-500">{label}:</span>
      <span className={cn('font-mono', color)}>{formatted}</span>
      {suffix && <span className="text-zinc-600 text-xs">{suffix}</span>}
    </div>
  )
}

// Auth prompt (only client-side interactive component needed)
export function AuthPrompt({ onAuth }: { onAuth: (key: string) => void }) {
  const [key, setKey] = useState('')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (key.trim()) {
      localStorage.setItem('shinapi_mgmt_key', key.trim())
      onAuth(key.trim())
    }
  }

  return (
    <div className="min-h-screen bg-zinc-950 flex items-center justify-center p-4">
      <div className="w-full max-w-md animate-fade-in-up">
        <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6">
          <h1 className="text-xl font-bold text-white mb-2">ShinAPI Dashboard</h1>
          <p className="text-sm text-zinc-400 mb-6">Enter your management key to continue</p>
          
          <form onSubmit={handleSubmit}>
            <input
              type="password"
              value={key}
              onChange={(e) => setKey(e.target.value)}
              placeholder="Management Secret Key"
              className="w-full px-4 py-3 rounded-lg bg-zinc-800 border border-zinc-700 text-white placeholder-zinc-500 focus:outline-none focus:border-blue-500"
              autoFocus
            />
            <button
              type="submit"
              className="w-full mt-4 px-4 py-3 rounded-lg bg-blue-600 hover:bg-blue-500 text-white font-medium transition-colors"
            >
              Access Dashboard
            </button>
          </form>
        </div>
      </div>
    </div>
  )
}
