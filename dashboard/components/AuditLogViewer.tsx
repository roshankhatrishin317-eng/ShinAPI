'use client'

import React, { useState, useEffect, useCallback, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { 
  Terminal, Filter, Download, Trash2, RefreshCw,
  Clock, Zap, AlertCircle, CheckCircle, ChevronDown,
  Search, X, Activity, Cpu, ArrowUpRight, ArrowDownRight,
  Circle, Minus, Square, Copy, ExternalLink, Play
} from 'lucide-react'
import { Badge } from '@/components/ui/badge'

interface AuditEntry {
  id: string
  timestamp: string
  level: 'info' | 'warning' | 'error' | 'debug'
  provider: string
  model: string
  auth_id?: string
  auth_label?: string
  endpoint: string
  method: string
  status_code: number
  latency_ms: number
  input_tokens?: number
  output_tokens?: number
  error?: string
  streaming: boolean
  cached: boolean
  request_body?: string
  response_preview?: string
}

interface AuditStats {
  total_entries: number
  error_count: number
  total_tokens: number
  avg_latency_ms: number
  provider_counts: Record<string, number>
  model_counts: Record<string, number>
  status_counts: Record<string, number>
  level_counts: Record<string, number>
}

interface FilterState {
  level: string
  provider: string
  model: string
  errorsOnly: boolean
  search: string
}

// macOS Window Controls
function MacOSControls({ onClose, onMinimize, onMaximize }: { 
  onClose?: () => void
  onMinimize?: () => void
  onMaximize?: () => void 
}) {
  return (
    <div className="flex items-center gap-2">
      <button 
        onClick={onClose}
        className="w-3 h-3 rounded-full bg-[#ff5f57] hover:bg-[#ff5f57]/80 transition-colors flex items-center justify-center group"
      >
        <X className="w-2 h-2 text-[#990000] opacity-0 group-hover:opacity-100" />
      </button>
      <button 
        onClick={onMinimize}
        className="w-3 h-3 rounded-full bg-[#febc2e] hover:bg-[#febc2e]/80 transition-colors flex items-center justify-center group"
      >
        <Minus className="w-2 h-2 text-[#995700] opacity-0 group-hover:opacity-100" />
      </button>
      <button 
        onClick={onMaximize}
        className="w-3 h-3 rounded-full bg-[#28c840] hover:bg-[#28c840]/80 transition-colors flex items-center justify-center group"
      >
        <Square className="w-1.5 h-1.5 text-[#006500] opacity-0 group-hover:opacity-100" />
      </button>
    </div>
  )
}

// Terminal Line Component
function TerminalLine({ 
  entry, 
  isSelected, 
  onClick,
  index 
}: { 
  entry: AuditEntry
  isSelected: boolean
  onClick: () => void
  index: number
}) {
  const getStatusColor = (code: number) => {
    if (code >= 500) return 'text-red-400'
    if (code >= 400) return 'text-amber-400'
    if (code >= 200 && code < 300) return 'text-green-400'
    return 'text-zinc-400'
  }

  const getLatencyColor = (ms: number) => {
    if (ms > 5000) return 'text-red-400'
    if (ms > 2000) return 'text-amber-400'
    if (ms > 500) return 'text-yellow-400'
    return 'text-green-400'
  }

  const formatTime = (ts: string) => {
    const d = new Date(ts)
    return d.toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' })
  }

  const totalTokens = (entry.input_tokens || 0) + (entry.output_tokens || 0)

  return (
    <motion.div
      initial={{ opacity: 0, x: -10 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ delay: index * 0.02 }}
      onClick={onClick}
      className={`font-mono text-xs cursor-pointer px-3 py-1.5 hover:bg-white/5 transition-colors ${
        isSelected ? 'bg-blue-500/20 border-l-2 border-blue-400' : 'border-l-2 border-transparent'
      }`}
    >
      <span className="text-zinc-600">[{formatTime(entry.timestamp)}]</span>
      {' '}
      <span className={getStatusColor(entry.status_code)}>{entry.status_code || '---'}</span>
      {' '}
      <span className="text-cyan-400">{entry.method}</span>
      {' '}
      <span className="text-zinc-300">{entry.endpoint}</span>
      {' '}
      <span className="text-purple-400">{entry.model}</span>
      {' '}
      <span className={getLatencyColor(entry.latency_ms)}>{entry.latency_ms}ms</span>
      {' '}
      {totalTokens > 0 && (
        <span className="text-zinc-500">({totalTokens} tokens)</span>
      )}
      {entry.streaming && <span className="text-blue-400 ml-1">[stream]</span>}
      {entry.cached && <span className="text-emerald-400 ml-1">[cached]</span>}
      {entry.error && <span className="text-red-400 ml-1">[error]</span>}
    </motion.div>
  )
}

// Request Details Panel
function RequestDetailsPanel({ entry, onClose }: { entry: AuditEntry; onClose: () => void }) {
  const formatDate = (ts: string) => {
    const d = new Date(ts)
    return d.toLocaleString('en-US', { 
      year: 'numeric', month: 'short', day: 'numeric',
      hour: '2-digit', minute: '2-digit', second: '2-digit', 
      hour12: false 
    })
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: 20 }}
      className="mt-4"
    >
      {/* macOS-style detail window */}
      <div className="rounded-xl overflow-hidden border border-zinc-700 bg-[#1e1e1e]">
        {/* Title bar */}
        <div className="flex items-center justify-between px-4 py-2 bg-[#323232] border-b border-zinc-700">
          <div className="flex items-center gap-3">
            <MacOSControls onClose={onClose} />
            <span className="text-xs text-zinc-400 ml-2">Request Details — {entry.id.slice(0, 8)}</span>
          </div>
          <button
            onClick={() => copyToClipboard(entry.id)}
            className="p-1 hover:bg-white/10 rounded transition-colors"
          >
            <Copy className="w-3 h-3 text-zinc-500" />
          </button>
        </div>

        {/* Content */}
        <div className="p-4 space-y-4">
          {/* Summary Cards */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            {/* Model */}
            <div className="p-3 rounded-lg bg-zinc-900/50 border border-zinc-800">
              <div className="flex items-center gap-2 text-zinc-500 text-xs mb-1">
                <Cpu className="w-3 h-3" />
                Model
              </div>
              <div className="text-sm font-medium text-white truncate">{entry.model}</div>
              <div className="text-xs text-zinc-500 mt-0.5">{entry.provider}</div>
            </div>

            {/* Tokens */}
            <div className="p-3 rounded-lg bg-zinc-900/50 border border-zinc-800">
              <div className="flex items-center gap-2 text-zinc-500 text-xs mb-1">
                <Zap className="w-3 h-3" />
                Tokens
              </div>
              <div className="flex items-center gap-2">
                <div className="flex items-center text-sm">
                  <ArrowUpRight className="w-3 h-3 text-blue-400 mr-1" />
                  <span className="text-blue-400 font-medium">{entry.input_tokens || 0}</span>
                </div>
                <div className="flex items-center text-sm">
                  <ArrowDownRight className="w-3 h-3 text-green-400 mr-1" />
                  <span className="text-green-400 font-medium">{entry.output_tokens || 0}</span>
                </div>
              </div>
              <div className="text-xs text-zinc-500 mt-0.5">
                Total: {(entry.input_tokens || 0) + (entry.output_tokens || 0)}
              </div>
            </div>

            {/* Latency */}
            <div className="p-3 rounded-lg bg-zinc-900/50 border border-zinc-800">
              <div className="flex items-center gap-2 text-zinc-500 text-xs mb-1">
                <Clock className="w-3 h-3" />
                Latency
              </div>
              <div className={`text-sm font-medium ${
                entry.latency_ms > 5000 ? 'text-red-400' :
                entry.latency_ms > 2000 ? 'text-amber-400' :
                entry.latency_ms > 500 ? 'text-yellow-400' : 'text-green-400'
              }`}>
                {entry.latency_ms}ms
              </div>
              <div className="text-xs text-zinc-500 mt-0.5">
                {(entry.latency_ms / 1000).toFixed(2)}s
              </div>
            </div>

            {/* Status */}
            <div className="p-3 rounded-lg bg-zinc-900/50 border border-zinc-800">
              <div className="flex items-center gap-2 text-zinc-500 text-xs mb-1">
                <Activity className="w-3 h-3" />
                Status
              </div>
              <div className={`text-sm font-medium ${
                entry.status_code >= 500 ? 'text-red-400' :
                entry.status_code >= 400 ? 'text-amber-400' :
                entry.status_code >= 200 ? 'text-green-400' : 'text-zinc-400'
              }`}>
                {entry.status_code || 'N/A'}
              </div>
              <div className="text-xs text-zinc-500 mt-0.5">
                {entry.status_code >= 200 && entry.status_code < 300 ? 'Success' :
                 entry.status_code >= 400 && entry.status_code < 500 ? 'Client Error' :
                 entry.status_code >= 500 ? 'Server Error' : 'Unknown'}
              </div>
            </div>
          </div>

          {/* Request Info Table */}
          <div className="rounded-lg border border-zinc-800 overflow-hidden">
            <div className="bg-zinc-900/50 px-3 py-2 border-b border-zinc-800">
              <span className="text-xs font-medium text-zinc-400">Request Information</span>
            </div>
            <div className="divide-y divide-zinc-800">
              <div className="flex px-3 py-2">
                <span className="text-xs text-zinc-500 w-32">Timestamp</span>
                <span className="text-xs text-zinc-300 font-mono">{formatDate(entry.timestamp)}</span>
              </div>
              <div className="flex px-3 py-2">
                <span className="text-xs text-zinc-500 w-32">Request ID</span>
                <span className="text-xs text-zinc-300 font-mono">{entry.id}</span>
              </div>
              <div className="flex px-3 py-2">
                <span className="text-xs text-zinc-500 w-32">Endpoint</span>
                <span className="text-xs text-cyan-400 font-mono">{entry.method} {entry.endpoint}</span>
              </div>
              <div className="flex px-3 py-2">
                <span className="text-xs text-zinc-500 w-32">Provider</span>
                <span className="text-xs text-zinc-300">{entry.provider}</span>
              </div>
              <div className="flex px-3 py-2">
                <span className="text-xs text-zinc-500 w-32">Auth</span>
                <span className="text-xs text-zinc-300">{entry.auth_label || entry.auth_id || 'N/A'}</span>
              </div>
              <div className="flex px-3 py-2">
                <span className="text-xs text-zinc-500 w-32">Flags</span>
                <div className="flex gap-2">
                  {entry.streaming && (
                    <span className="text-xs px-1.5 py-0.5 rounded bg-blue-500/10 text-blue-400 border border-blue-500/20">
                      Streaming
                    </span>
                  )}
                  {entry.cached && (
                    <span className="text-xs px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                      Cached
                    </span>
                  )}
                  {!entry.streaming && !entry.cached && (
                    <span className="text-xs text-zinc-500">None</span>
                  )}
                </div>
              </div>
            </div>
          </div>

          {/* Error Section */}
          {entry.error && (
            <div className="rounded-lg border border-red-500/30 bg-red-500/5 overflow-hidden">
              <div className="bg-red-500/10 px-3 py-2 border-b border-red-500/20 flex items-center gap-2">
                <AlertCircle className="w-3 h-3 text-red-400" />
                <span className="text-xs font-medium text-red-400">Error Details</span>
              </div>
              <pre className="p-3 text-xs text-red-300 font-mono whitespace-pre-wrap overflow-x-auto">
                {entry.error}
              </pre>
            </div>
          )}
        </div>
      </div>
    </motion.div>
  )
}

export function AuditLogViewer() {
  const [entries, setEntries] = useState<AuditEntry[]>([])
  const [stats, setStats] = useState<AuditStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [filter, setFilter] = useState<FilterState>({
    level: '',
    provider: '',
    model: '',
    errorsOnly: false,
    search: '',
  })
  const [showFilters, setShowFilters] = useState(false)
  const [selectedEntry, setSelectedEntry] = useState<AuditEntry | null>(null)
  const [autoScroll, setAutoScroll] = useState(true)
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [refreshInterval, setRefreshInterval] = useState(3000)
  const [lastUpdated, setLastUpdated] = useState(new Date())
  const terminalRef = useRef<HTMLDivElement>(null)

  const fetchLogs = useCallback(async () => {
    setLoading(true)
    try {
      const mgmtKey = localStorage.getItem('shinapi_mgmt_key') || ''
      const params = new URLSearchParams()
      if (filter.level) params.set('level', filter.level)
      if (filter.provider) params.set('provider', filter.provider)
      if (filter.model) params.set('model', filter.model)
      if (filter.errorsOnly) params.set('errors_only', 'true')
      params.set('limit', '200')

      const res = await fetch(`/v0/management/audit/logs?${params}`, {
        headers: { 'Authorization': `Bearer ${mgmtKey}` },
      })
      const data = await res.json()
      setEntries(data.entries || [])
    } catch (err) {
      console.error('Failed to fetch audit logs:', err)
      // Demo data for when API is not available
      setEntries([
        {
          id: 'req_001',
          timestamp: new Date().toISOString(),
          level: 'info',
          provider: 'openai',
          model: 'gpt-4-turbo',
          endpoint: '/v1/chat/completions',
          method: 'POST',
          status_code: 200,
          latency_ms: 1234,
          input_tokens: 150,
          output_tokens: 350,
          streaming: true,
          cached: false,
        },
        {
          id: 'req_002',
          timestamp: new Date(Date.now() - 60000).toISOString(),
          level: 'info',
          provider: 'claude',
          model: 'claude-3-opus',
          endpoint: '/v1/messages',
          method: 'POST',
          status_code: 200,
          latency_ms: 2567,
          input_tokens: 200,
          output_tokens: 800,
          streaming: false,
          cached: true,
        },
        {
          id: 'req_003',
          timestamp: new Date(Date.now() - 120000).toISOString(),
          level: 'error',
          provider: 'gemini',
          model: 'gemini-pro',
          endpoint: '/v1beta/models/gemini-pro:generateContent',
          method: 'POST',
          status_code: 429,
          latency_ms: 89,
          input_tokens: 50,
          output_tokens: 0,
          streaming: false,
          cached: false,
          error: 'Rate limit exceeded. Please try again later.',
        },
      ])
    } finally {
      setLoading(false)
    }
  }, [filter])

  const fetchStats = useCallback(async () => {
    try {
      const mgmtKey = localStorage.getItem('shinapi_mgmt_key') || ''
      const res = await fetch('/v0/management/audit/stats', {
        headers: { 'Authorization': `Bearer ${mgmtKey}` },
      })
      const data = await res.json()
      setStats(data)
    } catch (err) {
      // Demo stats
      setStats({
        total_entries: 1547,
        error_count: 23,
        total_tokens: 2456789,
        avg_latency_ms: 856,
        provider_counts: { openai: 800, claude: 500, gemini: 247 },
        model_counts: {},
        status_counts: { '200': 1500, '429': 20, '500': 3 },
        level_counts: { info: 1520, error: 23, warning: 4 },
      })
    }
  }, [])

  useEffect(() => {
    fetchLogs()
    fetchStats()
  }, [fetchLogs, fetchStats])

  useEffect(() => {
    if (autoRefresh) {
      const interval = setInterval(() => {
        fetchLogs()
        fetchStats()
        setLastUpdated(new Date())
      }, refreshInterval)
      return () => clearInterval(interval)
    }
  }, [autoRefresh, refreshInterval, fetchLogs, fetchStats])

  useEffect(() => {
    if (autoScroll && terminalRef.current) {
      terminalRef.current.scrollTop = 0
    }
  }, [entries, autoScroll])

  const handleClearLogs = useCallback(async () => {
    if (!confirm('Are you sure you want to clear all audit logs?')) return
    try {
      const mgmtKey = localStorage.getItem('shinapi_mgmt_key') || ''
      await fetch('/v0/management/audit/logs', {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${mgmtKey}` },
      })
      fetchLogs()
      fetchStats()
    } catch (err) {
      console.error('Failed to clear logs:', err)
    }
  }, [fetchLogs, fetchStats])

  const handleExport = useCallback(async () => {
    try {
      const mgmtKey = localStorage.getItem('shinapi_mgmt_key') || ''
      const res = await fetch('/v0/management/audit/export', {
        headers: { 'Authorization': `Bearer ${mgmtKey}` },
      })
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `audit-logs-${new Date().toISOString().split('T')[0]}.json`
      a.click()
      URL.revokeObjectURL(url)
    } catch (err) {
      console.error('Failed to export logs:', err)
    }
  }, [])

  const filteredEntries = entries.filter(entry => {
    if (filter.search) {
      const search = filter.search.toLowerCase()
      return (
        entry.model.toLowerCase().includes(search) ||
        entry.provider.toLowerCase().includes(search) ||
        entry.endpoint.toLowerCase().includes(search) ||
        entry.error?.toLowerCase().includes(search)
      )
    }
    return true
  })

  return (
    <div className="space-y-6">
      {/* Stats Row with Enhanced Design */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <motion.div 
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          whileHover={{ scale: 1.02, y: -2 }}
          className="relative p-4 rounded-2xl bg-gradient-to-br from-blue-500/10 via-blue-600/5 to-transparent border border-blue-500/20 overflow-hidden group"
        >
          <div className="absolute inset-0 bg-gradient-to-br from-blue-500/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity" />
          <div className="relative z-10">
            <div className="flex items-center gap-2 mb-2">
              <div className="p-2 rounded-lg bg-blue-500/10">
                <Activity className="w-4 h-4 text-blue-400" />
              </div>
              <span className="text-xs text-zinc-400">Total Requests</span>
            </div>
            <motion.span 
              key={stats?.total_entries}
              initial={{ opacity: 0.5, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1 }}
              className="text-3xl font-bold text-white"
            >
              {stats?.total_entries?.toLocaleString() || 0}
            </motion.span>
          </div>
        </motion.div>

        <motion.div 
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.05 }}
          whileHover={{ scale: 1.02, y: -2 }}
          className="relative p-4 rounded-2xl bg-gradient-to-br from-red-500/10 via-red-600/5 to-transparent border border-red-500/20 overflow-hidden group"
        >
          <div className="absolute inset-0 bg-gradient-to-br from-red-500/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity" />
          <div className="relative z-10">
            <div className="flex items-center gap-2 mb-2">
              <div className="p-2 rounded-lg bg-red-500/10">
                <AlertCircle className="w-4 h-4 text-red-400" />
              </div>
              <span className="text-xs text-zinc-400">Errors</span>
              {(stats?.error_count || 0) > 0 && (
                <span className="ml-auto text-[10px] px-1.5 py-0.5 rounded-full bg-red-500/20 text-red-400">
                  {((stats?.error_count || 0) / Math.max(stats?.total_entries || 1, 1) * 100).toFixed(1)}%
                </span>
              )}
            </div>
            <motion.span 
              key={stats?.error_count}
              initial={{ opacity: 0.5, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1 }}
              className="text-3xl font-bold text-white"
            >
              {stats?.error_count || 0}
            </motion.span>
          </div>
        </motion.div>

        <motion.div 
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
          whileHover={{ scale: 1.02, y: -2 }}
          className="relative p-4 rounded-2xl bg-gradient-to-br from-purple-500/10 via-purple-600/5 to-transparent border border-purple-500/20 overflow-hidden group"
        >
          <div className="absolute inset-0 bg-gradient-to-br from-purple-500/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity" />
          <div className="relative z-10">
            <div className="flex items-center gap-2 mb-2">
              <div className="p-2 rounded-lg bg-purple-500/10">
                <Zap className="w-4 h-4 text-purple-400" />
              </div>
              <span className="text-xs text-zinc-400">Total Tokens</span>
            </div>
            <motion.span 
              key={stats?.total_tokens}
              initial={{ opacity: 0.5, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1 }}
              className="text-3xl font-bold text-white"
            >
              {stats?.total_tokens?.toLocaleString() || 0}
            </motion.span>
          </div>
        </motion.div>

        <motion.div 
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.15 }}
          whileHover={{ scale: 1.02, y: -2 }}
          className="relative p-4 rounded-2xl bg-gradient-to-br from-emerald-500/10 via-emerald-600/5 to-transparent border border-emerald-500/20 overflow-hidden group"
        >
          <div className="absolute inset-0 bg-gradient-to-br from-emerald-500/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity" />
          <div className="relative z-10">
            <div className="flex items-center gap-2 mb-2">
              <div className="p-2 rounded-lg bg-emerald-500/10">
                <Clock className="w-4 h-4 text-emerald-400" />
              </div>
              <span className="text-xs text-zinc-400">Avg Latency</span>
            </div>
            <motion.span 
              key={stats?.avg_latency_ms}
              initial={{ opacity: 0.5, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1 }}
              className={`text-3xl font-bold ${
                (stats?.avg_latency_ms || 0) > 2000 ? 'text-amber-400' :
                (stats?.avg_latency_ms || 0) > 5000 ? 'text-red-400' : 'text-white'
              }`}
            >
              {stats?.avg_latency_ms || 0}
              <span className="text-lg text-zinc-500 ml-1">ms</span>
            </motion.span>
          </div>
        </motion.div>
      </div>

      {/* macOS Terminal Window */}
      <div className="rounded-xl overflow-hidden border border-zinc-700 shadow-2xl">
        {/* Title Bar */}
        <div className="flex items-center justify-between px-4 py-3 bg-gradient-to-b from-[#3a3a3a] to-[#2d2d2d] border-b border-zinc-700">
          <div className="flex items-center gap-4">
            <MacOSControls />
            <div className="flex items-center gap-2">
              <Terminal className="w-4 h-4 text-zinc-400" />
              <span className="text-sm text-zinc-300 font-medium">ShinAPI — Audit Logs</span>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setAutoRefresh(!autoRefresh)}
              className={`flex items-center gap-1.5 px-2 py-1 rounded text-xs transition-colors ${
                autoRefresh ? 'bg-green-500/20 text-green-400' : 'bg-zinc-700 text-zinc-400 hover:text-white'
              }`}
            >
              {autoRefresh ? (
                <>
                  <span className="relative flex h-2 w-2">
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                    <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500"></span>
                  </span>
                  Live
                </>
              ) : (
                <>
                  <Play className="w-3 h-3" />
                  Auto-refresh
                </>
              )}
            </button>
            <button
              onClick={() => setAutoScroll(!autoScroll)}
              className={`px-2 py-1 rounded text-xs transition-colors ${
                autoScroll ? 'bg-green-500/20 text-green-400' : 'bg-zinc-700 text-zinc-400'
              }`}
            >
              Auto-scroll
            </button>
            <button
              onClick={() => { fetchLogs(); fetchStats() }}
              className="p-1.5 rounded hover:bg-white/10 transition-colors text-zinc-400 hover:text-white"
              title="Refresh now"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            </button>
            <button
              onClick={handleExport}
              className="p-1.5 rounded hover:bg-white/10 transition-colors text-zinc-400 hover:text-white"
              title="Export logs"
            >
              <Download className="w-4 h-4" />
            </button>
            <button
              onClick={handleClearLogs}
              className="p-1.5 rounded hover:bg-red-500/20 transition-colors text-zinc-400 hover:text-red-400"
              title="Clear logs"
            >
              <Trash2 className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* Filter Bar */}
        <div className="flex items-center gap-3 px-4 py-2 bg-[#252525] border-b border-zinc-800">
          <div className="flex-1 relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-500" />
            <input
              type="text"
              value={filter.search}
              onChange={(e) => setFilter(prev => ({ ...prev, search: e.target.value }))}
              placeholder="Filter logs..."
              className="w-full pl-10 pr-4 py-1.5 rounded-md bg-zinc-900 border border-zinc-700 text-sm text-white placeholder-zinc-500 focus:border-blue-500/50 focus:outline-none font-mono"
            />
          </div>
          <select
            value={filter.level}
            onChange={(e) => setFilter(prev => ({ ...prev, level: e.target.value }))}
            className="px-2 py-1.5 rounded-md bg-zinc-900 border border-zinc-700 text-xs text-zinc-300 font-mono focus:outline-none focus:border-blue-500/50"
          >
            <option value="">All Levels</option>
            <option value="info">Info</option>
            <option value="warning">Warning</option>
            <option value="error">Error</option>
          </select>
          <select
            value={filter.provider}
            onChange={(e) => setFilter(prev => ({ ...prev, provider: e.target.value }))}
            className="px-2 py-1.5 rounded-md bg-zinc-900 border border-zinc-700 text-xs text-zinc-300 font-mono focus:outline-none focus:border-blue-500/50"
          >
            <option value="">All Providers</option>
            <option value="openai">OpenAI</option>
            <option value="claude">Claude</option>
            <option value="gemini">Gemini</option>
          </select>
          <label className="flex items-center gap-2 text-xs text-zinc-400 cursor-pointer hover:text-white transition-colors">
            <input
              type="checkbox"
              checked={filter.errorsOnly}
              onChange={(e) => setFilter(prev => ({ ...prev, errorsOnly: e.target.checked }))}
              className="rounded border-zinc-600 bg-zinc-900 focus:ring-0 focus:ring-offset-0"
            />
            Errors only
          </label>
        </div>

        {/* Terminal Content */}
        <div 
          ref={terminalRef}
          className="bg-[#1a1a1a] min-h-[400px] max-h-[600px] overflow-y-auto custom-scrollbar"
        >
          {loading && !entries.length ? (
            <div className="flex items-center justify-center py-20">
              <RefreshCw className="w-6 h-6 text-zinc-500 animate-spin" />
            </div>
          ) : filteredEntries.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-20 text-zinc-600">
              <Terminal className="w-12 h-12 mb-3 opacity-50" />
              <p className="font-mono text-sm">No logs found</p>
              <p className="font-mono text-xs mt-1">Waiting for requests...</p>
            </div>
          ) : (
            <div className="py-2">
              {/* Terminal prompt header */}
              <div className="px-3 py-1 font-mono text-xs text-zinc-500 border-b border-zinc-800 mb-2">
                $ tail -f /var/log/shinapi/audit.log | head -n {filteredEntries.length}
              </div>
              
              {filteredEntries.map((entry, idx) => (
                <TerminalLine
                  key={entry.id}
                  entry={entry}
                  index={idx}
                  isSelected={selectedEntry?.id === entry.id}
                  onClick={() => setSelectedEntry(selectedEntry?.id === entry.id ? null : entry)}
                />
              ))}
              
              {/* Blinking cursor */}
              <div className="px-3 py-1 font-mono text-xs">
                <span className="text-green-400">$</span>
                <span className="animate-pulse text-white ml-1">▋</span>
              </div>
            </div>
          )}
        </div>

        {/* Status Bar */}
        <div className="flex items-center justify-between px-4 py-2 bg-gradient-to-r from-[#252525] to-[#1f1f1f] border-t border-zinc-800 text-xs font-mono">
          <div className="flex items-center gap-4 text-zinc-500">
            <span className="flex items-center gap-1.5">
              <span className="w-1.5 h-1.5 rounded-full bg-zinc-500" />
              {filteredEntries.length} entries
            </span>
            <span className="text-zinc-700">|</span>
            <span className="flex items-center gap-1.5 text-green-400">
              <CheckCircle className="w-3 h-3" />
              {stats?.total_entries ? stats.total_entries - (stats?.error_count || 0) : 0} success
            </span>
            <span className="flex items-center gap-1.5 text-red-400">
              <AlertCircle className="w-3 h-3" />
              {stats?.error_count || 0} errors
            </span>
          </div>
          <div className="flex items-center gap-3 text-zinc-600">
            {autoRefresh && (
              <span className="flex items-center gap-1.5 text-green-400/70">
                <span className="relative flex h-1.5 w-1.5">
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
                  <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-green-500" />
                </span>
                Auto-refreshing every {refreshInterval / 1000}s
              </span>
            )}
            <span>Last updated: {lastUpdated.toLocaleTimeString()}</span>
          </div>
        </div>
      </div>

      {/* Selected Entry Details */}
      <AnimatePresence>
        {selectedEntry && (
          <RequestDetailsPanel 
            entry={selectedEntry} 
            onClose={() => setSelectedEntry(null)} 
          />
        )}
      </AnimatePresence>
    </div>
  )
}

export default AuditLogViewer
