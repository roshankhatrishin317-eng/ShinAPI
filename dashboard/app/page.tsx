'use client'

import React, { useState, useEffect, useCallback, useMemo } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { 
  Activity, Zap, Clock, TrendingUp, Database, Cpu, 
  Settings, BarChart3, PieChart as PieChartIcon, Timer, 
  Layers, AlertTriangle, Command, Keyboard, Sparkles,
  Radio, Server, RefreshCw, Moon, Sun, Bell, MessageSquare, FileText,
  Users, Globe, Shield, Gauge
} from 'lucide-react'
import { RingProgress } from '@mantine/core'

// Tremor components
import { MetricCard, MetricsGrid, CircularMetric } from '@/components/tremor/metrics'
import { 
  KPICard, 
  AreaChartComponent, 
  BarChartComponent, 
  DonutChartComponent,
  CategoryBarComponent 
} from '@/components/tremor/charts'

// Aceternity UI effects
import { SpotlightCard } from '@/components/ui/aceternity/spotlight'
import { BackgroundBeams, BackgroundGradientAnimation } from '@/components/ui/aceternity/background-beams'

// Live Metrics components
import { LiveMetricCard, LiveMetricsGrid, CompactStatCard } from '@/components/LiveMetrics'
import { UsageGraphs } from '@/components/UsageGraphs'

// Existing components
import { BentoCard, BentoGrid } from '@/components/ui/bento-card'
import { StatCard } from '@/components/ui/stat-card'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { useMetricsWebSocket } from '@/hooks/useWebSocket'
import { ConnectionStatus } from '@/components/ConnectionStatus'
import { RequestPipeline } from '@/components/RequestPipeline'
import { LatencyGrid } from '@/components/LatencyMeter'
import { ActivityFeed } from '@/components/ActivityFeed'
import { ErrorsPanel } from '@/components/ErrorsPanel'
import { ModelPieChart, RequestChart } from '@/components/Charts'
import { CommandPalette, useCommandPalette } from '@/components/CommandPalette'
import { APIPlayground } from '@/components/APIPlayground'
import { AuditLogViewer } from '@/components/AuditLogViewer'

interface Metrics {
  rpm: number
  tpm: number
  tps: number
  total_requests: number
  total_tokens: number
  total_success: number
  total_failed: number
  success_rate: number
  avg_latency_ms: number
  p50_latency_ms: number
  p95_latency_ms: number
  p99_latency_ms: number
  uptime_seconds: number
  model_stats: Record<string, { requests: number; tokens: number }>
  timestamp: number
  recent_requests?: any[]
  recent_errors?: any[]
}

function formatNumber(num: number): string {
  if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
  return Math.round(num).toString()
}

function formatDuration(seconds: number): string {
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return `${d}d ${h}h ${m}m`
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}

function Header({ 
  connectionState, 
  reconnectAttempts, 
  lastMessageTime,
  onOpenCommand,
  onLogout
}: {
  connectionState: string
  reconnectAttempts: number
  lastMessageTime: number | null
  onOpenCommand: () => void
  onLogout: () => void
}) {
  return (
    <header className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-8">
      <motion.div 
        className="flex items-center gap-4"
        initial={{ opacity: 0, x: -20 }}
        animate={{ opacity: 1, x: 0 }}
      >
        <motion.div 
          className="relative"
          whileHover={{ scale: 1.05 }}
          whileTap={{ scale: 0.95 }}
        >
          <div className="w-14 h-14 rounded-2xl bg-gradient-to-br from-blue-500 via-purple-500 to-pink-500 flex items-center justify-center font-bold text-2xl shadow-2xl shadow-purple-500/30">
            S
          </div>
          <motion.div
            className="absolute inset-0 rounded-2xl bg-gradient-to-br from-blue-500 via-purple-500 to-pink-500"
            animate={{ scale: [1, 1.2, 1], opacity: [0.5, 0, 0.5] }}
            transition={{ duration: 3, repeat: Infinity }}
          />
        </motion.div>
        
        <div>
          <h1 className="text-2xl font-bold gradient-text">
            ShinAPI Dashboard
          </h1>
          <p className="text-sm text-zinc-500 flex items-center gap-2">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
            Real-time metrics streaming
          </p>
        </div>
      </motion.div>

      <motion.div 
        className="flex items-center gap-3"
        initial={{ opacity: 0, x: 20 }}
        animate={{ opacity: 1, x: 0 }}
      >
        <button
          onClick={onOpenCommand}
          className="hidden sm:flex items-center gap-2 px-3 py-1.5 rounded-lg bg-zinc-800/50 border border-zinc-700/50 text-zinc-400 hover:text-white hover:border-zinc-600 transition-all text-sm"
        >
          <Command className="w-3.5 h-3.5" />
          <span>Command</span>
          <kbd className="px-1.5 py-0.5 rounded bg-zinc-700/50 text-[10px] font-mono">⌘K</kbd>
        </button>

        <ConnectionStatus 
          state={connectionState as any}
          reconnectAttempts={reconnectAttempts}
          lastMessageTime={lastMessageTime}
        />

        <Button 
          variant="ghost" 
          size="icon"
          onClick={onLogout}
          className="hover:bg-zinc-800"
        >
          <Settings className="w-4 h-4" />
        </Button>
      </motion.div>
    </header>
  )
}

function AuthScreen() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-zinc-950">
      <div className="fixed inset-0 -z-10">
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_center,rgba(59,130,246,0.15),transparent_60%)]" />
      </div>
      
      <motion.div
        initial={{ opacity: 0, scale: 0.9, y: 20 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        className="relative p-8 rounded-3xl bg-zinc-900/50 border border-white/10 backdrop-blur-xl max-w-md w-full mx-4"
      >
        <div className="absolute inset-0 rounded-3xl bg-gradient-to-br from-blue-500/10 via-transparent to-purple-500/10" />
        
        <div className="relative z-10 text-center">
          <motion.div 
            className="w-20 h-20 mx-auto mb-6 rounded-2xl bg-gradient-to-br from-blue-500 via-purple-500 to-pink-500 flex items-center justify-center text-3xl font-bold shadow-2xl shadow-purple-500/30"
            animate={{ rotate: [0, 5, -5, 0] }}
            transition={{ duration: 4, repeat: Infinity }}
          >
            S
          </motion.div>
          
          <h2 className="text-2xl font-bold text-white mb-2">Welcome to ShinAPI</h2>
          <p className="text-zinc-400 mb-6">Enter your management key to access the dashboard</p>
          
          <Button 
            onClick={() => window.location.reload()} 
            className="w-full bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700"
          >
            <RefreshCw className="w-4 h-4 mr-2" />
            Retry Authentication
          </Button>
        </div>
      </motion.div>
    </div>
  )
}

function FooterStats({ metrics }: { metrics: Metrics | null }) {
  const stats = [
    { label: 'Uptime', value: formatDuration(metrics?.uptime_seconds || 0) },
    { label: 'Total Requests', value: formatNumber(metrics?.total_requests || 0) },
    { label: 'Total Tokens', value: formatNumber(metrics?.total_tokens || 0) },
    { label: 'Success', value: formatNumber(metrics?.total_success || 0) },
    { label: 'Failed', value: formatNumber(metrics?.total_failed || 0) },
  ]

  return (
    <motion.footer 
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ delay: 0.5 }}
      className="mt-8 pt-6 border-t border-white/5"
    >
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex flex-wrap gap-6">
          {stats.map((stat, i) => (
            <motion.div 
              key={stat.label}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.6 + i * 0.05 }}
              className="text-sm"
            >
              <span className="text-zinc-500">{stat.label}: </span>
              <span className="text-zinc-200 font-medium">{stat.value}</span>
            </motion.div>
          ))}
        </div>
        
        <p className="text-xs text-zinc-600">
          Last updated: {new Date().toLocaleTimeString()}
        </p>
      </div>
    </motion.footer>
  )
}

export default function Dashboard() {
  const [apiKey, setApiKey] = useState<string | null>(null)
  const [metrics, setMetrics] = useState<Metrics | null>(null)
  const [history, setHistory] = useState<{ time: string; rpm: number; tpm: number; tps: number }[]>([])
  const [rpmHistory, setRpmHistory] = useState<number[]>([])
  const [tpmHistory, setTpmHistory] = useState<number[]>([])
  const [tpsHistory, setTpsHistory] = useState<number[]>([])
  const [prevMetrics, setPrevMetrics] = useState<Metrics | null>(null)
  const [activeTab, setActiveTab] = useState<'metrics' | 'playground' | 'logs'>('metrics')
  
  const commandPalette = useCommandPalette()

  useEffect(() => {
    let key = localStorage.getItem('shinapi_mgmt_key')
    if (!key) {
      key = prompt('Enter Management Secret Key:')
      if (key) localStorage.setItem('shinapi_mgmt_key', key)
    }
    setApiKey(key)
  }, [])

  const handleMessage = useCallback((data: Metrics) => {
    setMetrics(prev => {
      setPrevMetrics(prev)
      return data
    })
    
    setHistory(prev => {
      const entry = {
        time: new Date().toLocaleTimeString('en-US', { 
          hour12: false, 
          hour: '2-digit', 
          minute: '2-digit', 
          second: '2-digit' 
        }),
        rpm: data.rpm || 0,
        tpm: Math.round((data.tpm || 0) / 100),
        tps: data.tps || 0
      }
      return [...prev, entry].slice(-120)
    })
    
    setRpmHistory(prev => [...prev, data.rpm || 0].slice(-30))
    setTpmHistory(prev => [...prev, data.tpm || 0].slice(-30))
    setTpsHistory(prev => [...prev, data.tps || 0].slice(-30))
  }, [])

  const wsUrl = useMemo(() => {
    if (typeof window === 'undefined') return ''
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${protocol}//${window.location.host}/ws/metrics`
  }, [])

  const { connectionState, reconnectAttempts, lastMessageTime } = useMetricsWebSocket({
    url: wsUrl,
    authKey: apiKey,
    onMessage: handleMessage
  })

  const handleCommand = useCallback((cmd: string) => {
    switch (cmd) {
      case 'refresh':
        window.location.reload()
        break
      case 'settings':
        localStorage.removeItem('shinapi_mgmt_key')
        window.location.reload()
        break
    }
  }, [])

  // Safe trend calculation avoiding NaN/Infinity
  const calculateTrend = (current: number, previous: number): number => {
    if (!previous || previous === 0) return 0
    const trend = ((current - previous) / previous) * 100
    if (!isFinite(trend)) return 0
    return Math.max(-100, Math.min(100, trend)) // Clamp to reasonable range
  }

  const rpmTrend = calculateTrend(metrics?.rpm || 0, prevMetrics?.rpm || 0)
  const tpmTrend = calculateTrend(metrics?.tpm || 0, prevMetrics?.tpm || 0)
  const tpsTrend = calculateTrend(metrics?.tps || 0, prevMetrics?.tps || 0)

  const modelData = useMemo(() => {
    if (!metrics?.model_stats) return []
    return Object.entries(metrics.model_stats)
      .map(([name, stats]) => ({ name, ...stats }))
      .sort((a, b) => b.requests - a.requests)
  }, [metrics?.model_stats])

  if (!apiKey) return <AuthScreen />

  return (
    <div className="min-h-screen bg-zinc-950 text-white">
      {/* Animated background */}
      <div className="fixed inset-0 -z-10 overflow-hidden">
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top_left,rgba(59,130,246,0.12),transparent_50%)]" />
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_bottom_right,rgba(139,92,246,0.12),transparent_50%)]" />
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_center,rgba(6,182,212,0.05),transparent_70%)]" />
        
        {/* Grid pattern */}
        <svg className="absolute inset-0 w-full h-full opacity-[0.015]">
          <defs>
            <pattern id="grid" width="60" height="60" patternUnits="userSpaceOnUse">
              <path d="M 60 0 L 0 0 0 60" fill="none" stroke="white" strokeWidth="0.5"/>
            </pattern>
          </defs>
          <rect width="100%" height="100%" fill="url(#grid)" />
        </svg>
        
        {/* Noise texture */}
        <div className="absolute inset-0 noise" />
      </div>

      <AnimatePresence>
        <CommandPalette 
          isOpen={commandPalette.isOpen}
          onClose={commandPalette.close}
          onCommand={handleCommand}
        />
      </AnimatePresence>

      <div className="max-w-[1920px] mx-auto p-4 lg:p-8">
        <Header
          connectionState={connectionState}
          reconnectAttempts={reconnectAttempts}
          lastMessageTime={lastMessageTime}
          onOpenCommand={commandPalette.open}
          onLogout={() => {
            localStorage.removeItem('shinapi_mgmt_key')
            window.location.reload()
          }}
        />

        {/* Tab Navigation */}
        <div className="flex items-center gap-2 mb-6">
          <button
            onClick={() => setActiveTab('metrics')}
            className={`flex items-center gap-2 px-4 py-2.5 rounded-xl font-medium transition-all ${
              activeTab === 'metrics'
                ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30'
                : 'bg-zinc-900/50 text-zinc-400 border border-white/5 hover:text-white hover:border-white/10'
            }`}
          >
            <BarChart3 className="w-4 h-4" />
            Metrics
          </button>
          <button
            onClick={() => setActiveTab('playground')}
            className={`flex items-center gap-2 px-4 py-2.5 rounded-xl font-medium transition-all ${
              activeTab === 'playground'
                ? 'bg-purple-500/20 text-purple-400 border border-purple-500/30'
                : 'bg-zinc-900/50 text-zinc-400 border border-white/5 hover:text-white hover:border-white/10'
            }`}
          >
            <MessageSquare className="w-4 h-4" />
            Playground
          </button>
          <button
            onClick={() => setActiveTab('logs')}
            className={`flex items-center gap-2 px-4 py-2.5 rounded-xl font-medium transition-all ${
              activeTab === 'logs'
                ? 'bg-amber-500/20 text-amber-400 border border-amber-500/30'
                : 'bg-zinc-900/50 text-zinc-400 border border-white/5 hover:text-white hover:border-white/10'
            }`}
          >
            <FileText className="w-4 h-4" />
            Audit Logs
          </button>
        </div>

        {/* Tab Content */}
        {activeTab === 'metrics' && (
          <>
        {/* Hero Stats with Live Real-Time Graphs */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="mb-6"
        >
          <LiveMetricsGrid>
            <LiveMetricCard
              title="Requests/Min"
              value={metrics?.rpm || 0}
              icon={TrendingUp}
              color="blue"
              sparklineData={rpmHistory}
              trend={rpmTrend}
              showGraph={true}
            />
            <LiveMetricCard
              title="Tokens/Min"
              value={metrics?.tpm || 0}
              icon={Database}
              color="violet"
              sparklineData={tpmHistory}
              trend={tpmTrend}
              showGraph={true}
            />
            <LiveMetricCard
              title="Throughput"
              value={metrics?.tps || 0}
              suffix="/s"
              icon={Zap}
              color="cyan"
              sparklineData={tpsHistory}
              trend={tpsTrend}
              showGraph={true}
            />
            <LiveMetricCard
              title="Success Rate"
              value={metrics?.success_rate || 0}
              suffix="%"
              icon={Shield}
              color="emerald"
              sparklineData={[]}
              format="decimal"
              showGraph={false}
            />
          </LiveMetricsGrid>
        </motion.div>

        {/* Pipeline with Spotlight Effect */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
        >
          <SpotlightCard className="mb-6" spotlightColor="rgba(139, 92, 246, 0.15)">
            <div className="flex items-center gap-2 mb-4">
              <Layers className="w-5 h-5 text-purple-400" />
              <span className="text-sm font-semibold text-zinc-300">Request Pipeline</span>
              <Badge variant="purple" className="ml-auto">LIVE</Badge>
            </div>
            <RequestPipeline
              rpm={metrics?.rpm || 0}
              tps={metrics?.tps || 0}
              tpm={metrics?.tpm || 0}
              totalSuccess={metrics?.total_success || 0}
              totalFailed={metrics?.total_failed || 0}
            />
          </SpotlightCard>
        </motion.div>

        {/* Secondary Stats Row with Compact Cards */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          <CompactStatCard
            title="Total Requests"
            value={formatNumber(metrics?.total_requests || 0)}
            icon={Activity}
            color="blue"
          />
          <CompactStatCard
            title="Total Tokens"
            value={formatNumber(metrics?.total_tokens || 0)}
            icon={Database}
            color="violet"
          />
          <CompactStatCard
            title="Successful"
            value={formatNumber(metrics?.total_success || 0)}
            icon={Shield}
            color="emerald"
            trend={metrics?.total_requests ? (metrics.total_success / metrics.total_requests * 100) : 0}
          />
          <CompactStatCard
            title="Failed"
            value={formatNumber(metrics?.total_failed || 0)}
            icon={AlertTriangle}
            color="rose"
          />
        </div>

        {/* Usage Graphs Section */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
          className="mb-6"
        >
          <UsageGraphs
            rpmHistory={rpmHistory}
            tpmHistory={tpmHistory}
            tpsHistory={tpsHistory}
            totalTokens={metrics?.total_tokens || 0}
            avgLatency={metrics?.avg_latency_ms || 0}
          />
        </motion.div>

        {/* Charts Row */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-6">
          {/* Main Chart */}
          <BentoCard className="lg:col-span-2" glow="blue" size="md">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2">
                <BarChart3 className="w-5 h-5 text-blue-400" />
                <span className="text-sm font-semibold text-zinc-300">Request History</span>
              </div>
              <Badge variant="info">{history.length} points</Badge>
            </div>
            <RequestChart data={history} height={220} />
          </BentoCard>

          {/* Latency */}
          <BentoCard glow="amber" size="md">
            <div className="flex items-center gap-2 mb-4">
              <Timer className="w-5 h-5 text-amber-400" />
              <span className="text-sm font-semibold text-zinc-300">Latency</span>
            </div>
            <LatencyGrid
              avg={metrics?.avg_latency_ms || 0}
              p50={metrics?.p50_latency_ms || 0}
              p95={metrics?.p95_latency_ms || 0}
              p99={metrics?.p99_latency_ms || 0}
            />
          </BentoCard>
        </div>

        {/* Activity Row */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
          {/* Live Activity */}
          <BentoCard glow="emerald" size="md">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2">
                <Activity className="w-5 h-5 text-emerald-400" />
                <span className="text-sm font-semibold text-zinc-300">Live Activity</span>
              </div>
              <motion.div 
                className="w-2 h-2 rounded-full bg-emerald-400"
                animate={{ opacity: [1, 0.3, 1] }}
                transition={{ duration: 1.5, repeat: Infinity }}
              />
            </div>
            <ActivityFeed items={metrics?.recent_requests || []} maxItems={8} />
          </BentoCard>

          {/* Errors */}
          <BentoCard glow="rose" size="md">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2">
                <AlertTriangle className="w-5 h-5 text-rose-400" />
                <span className="text-sm font-semibold text-zinc-300">Recent Errors</span>
              </div>
              {(metrics?.total_failed || 0) > 0 && (
                <Badge variant="error">{metrics?.total_failed}</Badge>
              )}
            </div>
            <ErrorsPanel errors={metrics?.recent_errors || []} maxItems={5} />
          </BentoCard>

          {/* Model Distribution */}
          <BentoCard glow="purple" size="md">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2">
                <PieChartIcon className="w-5 h-5 text-purple-400" />
                <span className="text-sm font-semibold text-zinc-300">Model Usage</span>
              </div>
              <Badge variant="purple">{modelData.length} models</Badge>
            </div>
            <div className="h-[280px]">
              <ModelPieChart data={modelData} />
            </div>
          </BentoCard>
        </div>

        <FooterStats metrics={metrics} />
          </>
        )}

        {activeTab === 'playground' && (
          <BentoCard glow="purple" size="lg">
            <APIPlayground />
          </BentoCard>
        )}

        {activeTab === 'logs' && (
          <BentoCard glow="amber" size="lg">
            <AuditLogViewer />
          </BentoCard>
        )}
      </div>
    </div>
  )
}
