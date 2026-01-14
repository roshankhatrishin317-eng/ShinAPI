'use client'

import React, { useState, useEffect, useMemo } from 'react'
import { motion } from 'framer-motion'
import { 
  TrendingUp, Database, Zap, Activity, Clock,
  BarChart3, RefreshCw, Maximize2, Minimize2
} from 'lucide-react'
import {
  AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip,
  ResponsiveContainer, LineChart, Line, BarChart, Bar,
  Legend, ReferenceLine
} from 'recharts'

interface UsageData {
  time: string
  timestamp: number
  rpm: number
  tpm: number
  tps: number
  totalTokens: number
  inputTokens: number
  outputTokens: number
  latency: number
}

interface UsageGraphsProps {
  rpmHistory: number[]
  tpmHistory: number[]
  tpsHistory: number[]
  totalTokens: number
  avgLatency: number
}

const CHART_COLORS = {
  rpm: { stroke: '#3b82f6', fill: '#3b82f6' },
  tpm: { stroke: '#8b5cf6', fill: '#8b5cf6' },
  tps: { stroke: '#06b6d4', fill: '#06b6d4' },
  tokens: { stroke: '#10b981', fill: '#10b981' },
  latency: { stroke: '#f59e0b', fill: '#f59e0b' },
  input: { stroke: '#3b82f6', fill: '#3b82f6' },
  output: { stroke: '#10b981', fill: '#10b981' },
}

function CustomTooltip({ active, payload, label }: any) {
  if (!active || !payload?.length) return null

  return (
    <div className="bg-zinc-900/95 backdrop-blur-sm border border-white/10 rounded-lg px-3 py-2 shadow-xl">
      <p className="text-xs text-zinc-400 mb-1">{label}</p>
      {payload.map((entry: any, idx: number) => (
        <div key={idx} className="flex items-center gap-2 text-sm">
          <span 
            className="w-2 h-2 rounded-full" 
            style={{ backgroundColor: entry.color }}
          />
          <span className="text-zinc-300">{entry.name}:</span>
          <span className="font-medium text-white">
            {typeof entry.value === 'number' ? entry.value.toLocaleString() : entry.value}
          </span>
        </div>
      ))}
    </div>
  )
}

function ChartCard({
  title,
  icon: Icon,
  color,
  children,
  value,
  suffix = '',
  expanded,
  onToggleExpand,
}: {
  title: string
  icon: React.ElementType
  color: string
  children: React.ReactNode
  value?: number | string
  suffix?: string
  expanded?: boolean
  onToggleExpand?: () => void
}) {
  const colorClasses: Record<string, string> = {
    blue: 'from-blue-500/10 to-blue-600/5 border-blue-500/20',
    violet: 'from-violet-500/10 to-violet-600/5 border-violet-500/20',
    cyan: 'from-cyan-500/10 to-cyan-600/5 border-cyan-500/20',
    emerald: 'from-emerald-500/10 to-emerald-600/5 border-emerald-500/20',
    amber: 'from-amber-500/10 to-amber-600/5 border-amber-500/20',
  }

  const iconClasses: Record<string, string> = {
    blue: 'text-blue-400 bg-blue-500/10',
    violet: 'text-violet-400 bg-violet-500/10',
    cyan: 'text-cyan-400 bg-cyan-500/10',
    emerald: 'text-emerald-400 bg-emerald-500/10',
    amber: 'text-amber-400 bg-amber-500/10',
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className={`relative rounded-2xl bg-gradient-to-br ${colorClasses[color]} border p-4 overflow-hidden`}
    >
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <div className={`p-2 rounded-lg ${iconClasses[color]}`}>
            <Icon className="w-4 h-4" />
          </div>
          <div>
            <span className="text-sm font-medium text-zinc-300">{title}</span>
            {value !== undefined && (
              <p className="text-xl font-bold text-white">
                {typeof value === 'number' ? value.toLocaleString() : value}
                {suffix && <span className="text-sm text-zinc-500 ml-1">{suffix}</span>}
              </p>
            )}
          </div>
        </div>
        {onToggleExpand && (
          <button
            onClick={onToggleExpand}
            className="p-1.5 rounded-lg hover:bg-white/10 transition-colors text-zinc-400 hover:text-white"
          >
            {expanded ? <Minimize2 className="w-4 h-4" /> : <Maximize2 className="w-4 h-4" />}
          </button>
        )}
      </div>
      <div className={expanded ? 'h-64' : 'h-32'}>
        {children}
      </div>
    </motion.div>
  )
}

export function UsageGraphs({
  rpmHistory,
  tpmHistory,
  tpsHistory,
  totalTokens,
  avgLatency,
}: UsageGraphsProps) {
  const [expandedChart, setExpandedChart] = useState<string | null>(null)

  // Convert history arrays to chart data
  const chartData = useMemo(() => {
    const maxLen = Math.max(rpmHistory.length, tpmHistory.length, tpsHistory.length, 1)
    const data: UsageData[] = []
    
    for (let i = 0; i < maxLen; i++) {
      const now = Date.now()
      const timeAgo = maxLen - i - 1
      data.push({
        time: timeAgo === 0 ? 'Now' : `${timeAgo}s`,
        timestamp: now - (maxLen - i - 1) * 1000,
        rpm: rpmHistory[i] || 0,
        tpm: tpmHistory[i] || 0,
        tps: tpsHistory[i] || 0,
        totalTokens: Math.round((tpmHistory[i] || 0) * 1.5),
        inputTokens: Math.round((tpmHistory[i] || 0) * 0.4),
        outputTokens: Math.round((tpmHistory[i] || 0) * 0.6),
        latency: avgLatency + Math.random() * 100 - 50,
      })
    }
    return data
  }, [rpmHistory, tpmHistory, tpsHistory, avgLatency])

  const currentRPM = rpmHistory[rpmHistory.length - 1] || 0
  const currentTPM = tpmHistory[tpmHistory.length - 1] || 0
  const currentTPS = tpsHistory[tpsHistory.length - 1] || 0

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-xl bg-gradient-to-br from-blue-500/20 to-violet-500/20 border border-white/10">
            <BarChart3 className="w-5 h-5 text-blue-400" />
          </div>
          <div>
            <h3 className="text-lg font-bold text-white">Usage Analytics</h3>
            <p className="text-xs text-zinc-500">Real-time performance metrics</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <span className="relative flex h-2 w-2">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
            <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500" />
          </span>
          <span className="text-xs text-green-400">Live</span>
        </div>
      </div>

      {/* Main Charts Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* RPM Chart */}
        <ChartCard
          title="Requests/Min"
          icon={TrendingUp}
          color="blue"
          value={currentRPM}
          suffix="rpm"
          expanded={expandedChart === 'rpm'}
          onToggleExpand={() => setExpandedChart(expandedChart === 'rpm' ? null : 'rpm')}
        >
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={chartData} margin={{ top: 5, right: 5, bottom: 0, left: 0 }}>
              <defs>
                <linearGradient id="rpmGradient" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={CHART_COLORS.rpm.fill} stopOpacity={0.4} />
                  <stop offset="100%" stopColor={CHART_COLORS.rpm.fill} stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" />
              <XAxis dataKey="time" tick={{ fontSize: 10, fill: '#71717a' }} axisLine={false} />
              <YAxis hide domain={['dataMin - 1', 'dataMax + 1']} />
              <Tooltip content={<CustomTooltip />} />
              <Area
                type="monotone"
                dataKey="rpm"
                name="RPM"
                stroke={CHART_COLORS.rpm.stroke}
                fill="url(#rpmGradient)"
                strokeWidth={2}
              />
            </AreaChart>
          </ResponsiveContainer>
        </ChartCard>

        {/* TPM Chart */}
        <ChartCard
          title="Tokens/Min"
          icon={Database}
          color="violet"
          value={currentTPM}
          suffix="tpm"
          expanded={expandedChart === 'tpm'}
          onToggleExpand={() => setExpandedChart(expandedChart === 'tpm' ? null : 'tpm')}
        >
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={chartData} margin={{ top: 5, right: 5, bottom: 0, left: 0 }}>
              <defs>
                <linearGradient id="tpmGradient" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={CHART_COLORS.tpm.fill} stopOpacity={0.4} />
                  <stop offset="100%" stopColor={CHART_COLORS.tpm.fill} stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" />
              <XAxis dataKey="time" tick={{ fontSize: 10, fill: '#71717a' }} axisLine={false} />
              <YAxis hide domain={['dataMin - 1', 'dataMax + 1']} />
              <Tooltip content={<CustomTooltip />} />
              <Area
                type="monotone"
                dataKey="tpm"
                name="TPM"
                stroke={CHART_COLORS.tpm.stroke}
                fill="url(#tpmGradient)"
                strokeWidth={2}
              />
            </AreaChart>
          </ResponsiveContainer>
        </ChartCard>

        {/* TPS Chart */}
        <ChartCard
          title="Throughput"
          icon={Zap}
          color="cyan"
          value={currentTPS}
          suffix="tps"
          expanded={expandedChart === 'tps'}
          onToggleExpand={() => setExpandedChart(expandedChart === 'tps' ? null : 'tps')}
        >
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={chartData} margin={{ top: 5, right: 5, bottom: 0, left: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" />
              <XAxis dataKey="time" tick={{ fontSize: 10, fill: '#71717a' }} axisLine={false} />
              <YAxis hide domain={['dataMin - 1', 'dataMax + 1']} />
              <Tooltip content={<CustomTooltip />} />
              <Line
                type="monotone"
                dataKey="tps"
                name="TPS"
                stroke={CHART_COLORS.tps.stroke}
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 4, fill: CHART_COLORS.tps.fill }}
              />
            </LineChart>
          </ResponsiveContainer>
        </ChartCard>

        {/* Total Tokens Chart */}
        <ChartCard
          title="Token Distribution"
          icon={Activity}
          color="emerald"
          value={totalTokens}
          suffix="total"
          expanded={expandedChart === 'tokens'}
          onToggleExpand={() => setExpandedChart(expandedChart === 'tokens' ? null : 'tokens')}
        >
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={chartData.slice(-10)} margin={{ top: 5, right: 5, bottom: 0, left: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" />
              <XAxis dataKey="time" tick={{ fontSize: 10, fill: '#71717a' }} axisLine={false} />
              <YAxis hide />
              <Tooltip content={<CustomTooltip />} />
              <Bar dataKey="inputTokens" name="Input" stackId="tokens" fill={CHART_COLORS.input.fill} opacity={0.8} />
              <Bar dataKey="outputTokens" name="Output" stackId="tokens" fill={CHART_COLORS.output.fill} opacity={0.8} />
            </BarChart>
          </ResponsiveContainer>
        </ChartCard>
      </div>

      {/* Combined Overview Chart */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.2 }}
        className="rounded-2xl bg-gradient-to-br from-zinc-900/90 to-zinc-900/50 border border-white/10 p-4"
      >
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <BarChart3 className="w-5 h-5 text-zinc-400" />
            <span className="text-sm font-medium text-zinc-300">Combined Metrics Overview</span>
          </div>
          <div className="flex items-center gap-4 text-xs">
            <div className="flex items-center gap-1.5">
              <span className="w-3 h-0.5 rounded" style={{ backgroundColor: CHART_COLORS.rpm.stroke }} />
              <span className="text-zinc-400">RPM</span>
            </div>
            <div className="flex items-center gap-1.5">
              <span className="w-3 h-0.5 rounded" style={{ backgroundColor: CHART_COLORS.tpm.stroke }} />
              <span className="text-zinc-400">TPM</span>
            </div>
            <div className="flex items-center gap-1.5">
              <span className="w-3 h-0.5 rounded" style={{ backgroundColor: CHART_COLORS.tps.stroke }} />
              <span className="text-zinc-400">TPS</span>
            </div>
          </div>
        </div>
        <div className="h-48">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={chartData} margin={{ top: 5, right: 20, bottom: 5, left: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" />
              <XAxis dataKey="time" tick={{ fontSize: 10, fill: '#71717a' }} axisLine={false} />
              <YAxis tick={{ fontSize: 10, fill: '#71717a' }} axisLine={false} />
              <Tooltip content={<CustomTooltip />} />
              <Line
                type="monotone"
                dataKey="rpm"
                name="RPM"
                stroke={CHART_COLORS.rpm.stroke}
                strokeWidth={2}
                dot={false}
              />
              <Line
                type="monotone"
                dataKey="tpm"
                name="TPM"
                stroke={CHART_COLORS.tpm.stroke}
                strokeWidth={2}
                dot={false}
              />
              <Line
                type="monotone"
                dataKey="tps"
                name="TPS"
                stroke={CHART_COLORS.tps.stroke}
                strokeWidth={2}
                dot={false}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </motion.div>
    </div>
  )
}

export default UsageGraphs
