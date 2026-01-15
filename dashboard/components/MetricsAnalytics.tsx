'use client'

import React, { useState, useMemo } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { 
  BarChart3, Download, Settings, RefreshCw, 
  Zap, Database, Clock, Calendar, TrendingUp,
  Grid3X3, LayoutGrid, Maximize2
} from 'lucide-react'
import { cn } from '@/lib/utils'

import { TPSChart } from './charts/TPSChart'
import { TPMChart } from './charts/TPMChart'
import { TPHChart } from './charts/TPHChart'
import { TPDChart } from './charts/TPDChart'
import { ThroughputMeter } from './ThroughputMeter'
import { useAllMetrics, useHistoricalMetrics } from '@/hooks/useHistoricalMetrics'
import { ShimmerButton } from './ui/magic/shimmer-button'
import { NumberTicker } from './ui/magic/number-ticker'
import { BorderBeam } from './ui/magic/border-beam'

type TimeRange = 'live' | '1h' | '24h' | '7d' | '30d'

interface MetricsAnalyticsProps {
  className?: string
}

const timeRanges: { value: TimeRange; label: string }[] = [
  { value: 'live', label: 'Live' },
  { value: '1h', label: '1 Hour' },
  { value: '24h', label: '24 Hours' },
  { value: '7d', label: '7 Days' },
  { value: '30d', label: '30 Days' },
]

function StatCard({ 
  label, 
  value, 
  suffix,
  icon: Icon,
  color = 'blue'
}: { 
  label: string
  value: number
  suffix?: string
  icon: React.ElementType
  color?: 'blue' | 'violet' | 'cyan' | 'emerald' | 'amber'
}) {
  const colorClasses = {
    blue: 'from-blue-500/20 to-blue-600/5 border-blue-500/30 text-blue-400',
    violet: 'from-violet-500/20 to-violet-600/5 border-violet-500/30 text-violet-400',
    cyan: 'from-cyan-500/20 to-cyan-600/5 border-cyan-500/30 text-cyan-400',
    emerald: 'from-emerald-500/20 to-emerald-600/5 border-emerald-500/30 text-emerald-400',
    amber: 'from-amber-500/20 to-amber-600/5 border-amber-500/30 text-amber-400',
  }

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      className={cn(
        'relative overflow-hidden rounded-xl border p-4',
        'bg-gradient-to-br',
        colorClasses[color]
      )}
    >
      <BorderBeam 
        size={100} 
        duration={10} 
        colorFrom={color === 'blue' ? '#3b82f6' : color === 'violet' ? '#8b5cf6' : color === 'cyan' ? '#06b6d4' : color === 'emerald' ? '#10b981' : '#f59e0b'}
        colorTo={color === 'blue' ? '#60a5fa' : color === 'violet' ? '#a78bfa' : color === 'cyan' ? '#22d3ee' : color === 'emerald' ? '#34d399' : '#fbbf24'}
      />
      <div className="relative z-10">
        <div className="flex items-center gap-2 mb-2">
          <Icon className="w-4 h-4" />
          <span className="text-xs text-zinc-400">{label}</span>
        </div>
        <div className="flex items-baseline gap-1">
          <NumberTicker 
            value={value} 
            className="text-2xl font-bold text-white"
            decimalPlaces={value < 100 ? 1 : 0}
          />
          {suffix && <span className="text-sm text-zinc-500">{suffix}</span>}
        </div>
      </div>
    </motion.div>
  )
}

export function MetricsAnalytics({ className }: MetricsAnalyticsProps) {
  const [timeRange, setTimeRange] = useState<TimeRange>('live')
  const [expandedChart, setExpandedChart] = useState<string | null>(null)
  const [isRefreshing, setIsRefreshing] = useState(false)

  const { tps, tpm, tph, tpd, isLoading } = useAllMetrics()
  const historical = useHistoricalMetrics(timeRange === 'live' ? '1h' : timeRange)

  const handleRefresh = async () => {
    setIsRefreshing(true)
    await Promise.all([
      tps.refresh(),
      tpm.refresh(),
      tph.refresh(),
      tpd.refresh(),
      historical.refresh(),
    ])
    setTimeout(() => setIsRefreshing(false), 500)
  }

  const handleExport = (chartType: string) => {
    let data: any[] = []
    let filename = ''

    switch (chartType) {
      case 'tps':
        data = tps.data
        filename = 'tps-data.json'
        break
      case 'tpm':
        data = tpm.data
        filename = 'tpm-data.json'
        break
      case 'tph':
        data = tph.data
        filename = 'tph-data.json'
        break
      case 'tpd':
        data = tpd.data
        filename = 'tpd-data.json'
        break
    }

    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div className={cn('space-y-6', className)}>
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <div className="p-2.5 rounded-xl bg-gradient-to-br from-blue-500/20 to-violet-500/20 border border-white/10">
            <BarChart3 className="w-5 h-5 text-blue-400" />
          </div>
          <div>
            <h2 className="text-xl font-bold text-white">Metrics Analytics</h2>
            <p className="text-sm text-zinc-500">Real-time performance monitoring with historical data</p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          {/* Time Range Selector */}
          <div className="flex items-center bg-zinc-900/50 rounded-xl border border-white/10 p-1">
            {timeRanges.map((range) => (
              <button
                key={range.value}
                onClick={() => setTimeRange(range.value)}
                className={cn(
                  'px-3 py-1.5 text-sm font-medium rounded-lg transition-all',
                  timeRange === range.value
                    ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30'
                    : 'text-zinc-400 hover:text-white hover:bg-white/5'
                )}
              >
                {range.label}
              </button>
            ))}
          </div>

          {/* Refresh Button */}
          <button
            onClick={handleRefresh}
            disabled={isRefreshing}
            className={cn(
              'p-2 rounded-xl bg-zinc-900/50 border border-white/10',
              'hover:bg-white/5 transition-colors',
              isRefreshing && 'animate-spin'
            )}
          >
            <RefreshCw className="w-4 h-4 text-zinc-400" />
          </button>
        </div>
      </div>

      {/* Quick Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard
          label="Current TPS"
          value={tps.currentTPS}
          suffix="/s"
          icon={Zap}
          color="cyan"
        />
        <StatCard
          label="Tokens/Min"
          value={tpm.currentTPM}
          suffix="TPM"
          icon={Database}
          color="violet"
        />
        <StatCard
          label="Tokens/Hour"
          value={tph.currentTPH}
          suffix="TPH"
          icon={Clock}
          color="amber"
        />
        <StatCard
          label="Tokens/Day"
          value={tpd.currentTPD}
          suffix="TPD"
          icon={Calendar}
          color="emerald"
        />
      </div>

      {/* Throughput Meter */}
      <ThroughputMeter
        tps={tps.currentTPS}
        tpm={tpm.currentTPM}
        tph={tph.currentTPH}
        tpd={tpd.currentTPD}
        maxTps={100}
        maxTpm={100000}
      />

      {/* Charts Grid */}
      <div className={cn(
        'grid gap-4',
        expandedChart ? 'grid-cols-1' : 'grid-cols-1 lg:grid-cols-2'
      )}>
        <AnimatePresence mode="popLayout">
          {(!expandedChart || expandedChart === 'tps') && (
            <motion.div
              key="tps"
              layout
              initial={{ opacity: 0, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.95 }}
            >
              <TPSChart
                data={tps.data}
                currentTPS={tps.currentTPS}
                expanded={expandedChart === 'tps'}
                onToggleExpand={() => setExpandedChart(expandedChart === 'tps' ? null : 'tps')}
                onExport={() => handleExport('tps')}
              />
            </motion.div>
          )}

          {(!expandedChart || expandedChart === 'tpm') && (
            <motion.div
              key="tpm"
              layout
              initial={{ opacity: 0, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.95 }}
            >
              <TPMChart
                data={tpm.data}
                currentTPM={tpm.currentTPM}
                expanded={expandedChart === 'tpm'}
                onToggleExpand={() => setExpandedChart(expandedChart === 'tpm' ? null : 'tpm')}
                onExport={() => handleExport('tpm')}
              />
            </motion.div>
          )}

          {(!expandedChart || expandedChart === 'tph') && (
            <motion.div
              key="tph"
              layout
              initial={{ opacity: 0, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.95 }}
            >
              <TPHChart
                data={tph.data}
                currentTPH={tph.currentTPH}
                expanded={expandedChart === 'tph'}
                onToggleExpand={() => setExpandedChart(expandedChart === 'tph' ? null : 'tph')}
                onExport={() => handleExport('tph')}
              />
            </motion.div>
          )}

          {(!expandedChart || expandedChart === 'tpd') && (
            <motion.div
              key="tpd"
              layout
              initial={{ opacity: 0, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.95 }}
            >
              <TPDChart
                data={tpd.data}
                currentTPD={tpd.currentTPD}
                expanded={expandedChart === 'tpd'}
                onToggleExpand={() => setExpandedChart(expandedChart === 'tpd' ? null : 'tpd')}
                onExport={() => handleExport('tpd')}
              />
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      {/* Summary Stats */}
      {historical.summary && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="relative overflow-hidden rounded-2xl bg-zinc-900/50 border border-white/10 p-6"
        >
          <div className="absolute inset-0 bg-gradient-to-br from-blue-500/5 via-transparent to-violet-500/5" />
          <div className="relative z-10">
            <h3 className="text-sm font-semibold text-zinc-300 mb-4">
              Summary ({historical.range})
            </h3>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
              <div>
                <p className="text-xs text-zinc-500 mb-1">Total Requests</p>
                <p className="text-xl font-bold text-white">
                  {historical.summary.total_requests.toLocaleString()}
                </p>
              </div>
              <div>
                <p className="text-xs text-zinc-500 mb-1">Total Tokens</p>
                <p className="text-xl font-bold text-white">
                  {historical.summary.total_tokens.toLocaleString()}
                </p>
              </div>
              <div>
                <p className="text-xs text-zinc-500 mb-1">Success Rate</p>
                <p className="text-xl font-bold text-emerald-400">
                  {historical.summary.success_rate.toFixed(1)}%
                </p>
              </div>
              <div>
                <p className="text-xs text-zinc-500 mb-1">Avg Latency</p>
                <p className="text-xl font-bold text-amber-400">
                  {historical.summary.avg_latency_ms.toFixed(0)}ms
                </p>
              </div>
            </div>
          </div>
        </motion.div>
      )}
    </div>
  )
}

export default MetricsAnalytics
