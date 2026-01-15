// Server Component - No 'use client' directive
// This renders on the server and sends only HTML to the browser

import { cn } from '@/lib/utils'
import { 
  Activity, Zap, Database, Clock, TrendingUp, 
  Shield, AlertTriangle, Server 
} from 'lucide-react'

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

interface MetricCardProps {
  title: string
  value: string | number
  suffix?: string
  icon: React.ElementType
  color: 'blue' | 'violet' | 'cyan' | 'emerald' | 'amber' | 'rose'
  trend?: number
  delay?: number
}

const colorClasses = {
  blue: {
    bg: 'from-blue-500/20 to-blue-600/5',
    border: 'border-blue-500/30',
    text: 'text-blue-400',
    glow: 'hover:shadow-blue-500/20',
  },
  violet: {
    bg: 'from-violet-500/20 to-violet-600/5',
    border: 'border-violet-500/30',
    text: 'text-violet-400',
    glow: 'hover:shadow-violet-500/20',
  },
  cyan: {
    bg: 'from-cyan-500/20 to-cyan-600/5',
    border: 'border-cyan-500/30',
    text: 'text-cyan-400',
    glow: 'hover:shadow-cyan-500/20',
  },
  emerald: {
    bg: 'from-emerald-500/20 to-emerald-600/5',
    border: 'border-emerald-500/30',
    text: 'text-emerald-400',
    glow: 'hover:shadow-emerald-500/20',
  },
  amber: {
    bg: 'from-amber-500/20 to-amber-600/5',
    border: 'border-amber-500/30',
    text: 'text-amber-400',
    glow: 'hover:shadow-amber-500/20',
  },
  rose: {
    bg: 'from-rose-500/20 to-rose-600/5',
    border: 'border-rose-500/30',
    text: 'text-rose-400',
    glow: 'hover:shadow-rose-500/20',
  },
}

function MetricCard({ 
  title, 
  value, 
  suffix, 
  icon: Icon, 
  color, 
  trend,
  delay = 0 
}: MetricCardProps) {
  const colors = colorClasses[color]
  const staggerClass = delay > 0 ? `stagger-${Math.min(delay, 8)}` : ''
  
  return (
    <div 
      className={cn(
        'relative overflow-hidden rounded-xl border p-4',
        'bg-gradient-to-br',
        colors.bg,
        colors.border,
        'hover-lift hover-glow',
        'animate-fade-in-up',
        staggerClass
      )}
    >
      {/* Shimmer effect */}
      <div className="absolute inset-0 animate-shimmer pointer-events-none" />
      
      <div className="relative z-10">
        <div className="flex items-center gap-2 mb-2">
          <Icon className={cn('w-4 h-4', colors.text)} />
          <span className="text-xs text-zinc-400">{title}</span>
        </div>
        
        <div className="flex items-baseline gap-1">
          <span className="text-2xl font-bold text-white animate-number-tick">
            {typeof value === 'number' ? formatNumber(value) : value}
          </span>
          {suffix && (
            <span className="text-sm text-zinc-500">{suffix}</span>
          )}
        </div>
        
        {trend !== undefined && (
          <div className={cn(
            'mt-2 text-xs flex items-center gap-1',
            trend >= 0 ? 'text-emerald-400' : 'text-rose-400'
          )}>
            <TrendingUp className={cn('w-3 h-3', trend < 0 && 'rotate-180')} />
            <span>{Math.abs(trend).toFixed(1)}%</span>
          </div>
        )}
      </div>
    </div>
  )
}

interface MetricsCardsProps {
  metrics: Metrics
}

export function MetricsCards({ metrics }: MetricsCardsProps) {
  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
      <MetricCard
        title="Requests/Min"
        value={metrics.rpm}
        icon={TrendingUp}
        color="blue"
        delay={1}
      />
      <MetricCard
        title="Tokens/Min"
        value={metrics.tpm}
        icon={Database}
        color="violet"
        delay={2}
      />
      <MetricCard
        title="Throughput"
        value={metrics.tps}
        suffix="/s"
        icon={Zap}
        color="cyan"
        delay={3}
      />
      <MetricCard
        title="Success Rate"
        value={metrics.success_rate?.toFixed(1) || '0'}
        suffix="%"
        icon={Shield}
        color="emerald"
        delay={4}
      />
    </div>
  )
}

export function StatsCards({ metrics }: MetricsCardsProps) {
  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
      <MetricCard
        title="Total Requests"
        value={metrics.total_requests}
        icon={Activity}
        color="blue"
        delay={1}
      />
      <MetricCard
        title="Total Tokens"
        value={metrics.total_tokens}
        icon={Database}
        color="violet"
        delay={2}
      />
      <MetricCard
        title="Successful"
        value={metrics.total_success}
        icon={Shield}
        color="emerald"
        trend={metrics.total_requests ? (metrics.total_success / metrics.total_requests * 100) : 0}
        delay={3}
      />
      <MetricCard
        title="Failed"
        value={metrics.total_failed}
        icon={AlertTriangle}
        color="rose"
        delay={4}
      />
    </div>
  )
}

export function LatencyCards({ metrics }: MetricsCardsProps) {
  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
      <MetricCard
        title="Avg Latency"
        value={Math.round(metrics.avg_latency_ms)}
        suffix="ms"
        icon={Clock}
        color="amber"
        delay={1}
      />
      <MetricCard
        title="P50 Latency"
        value={Math.round(metrics.p50_latency_ms)}
        suffix="ms"
        icon={Clock}
        color="blue"
        delay={2}
      />
      <MetricCard
        title="P95 Latency"
        value={Math.round(metrics.p95_latency_ms)}
        suffix="ms"
        icon={Clock}
        color="violet"
        delay={3}
      />
      <MetricCard
        title="P99 Latency"
        value={Math.round(metrics.p99_latency_ms)}
        suffix="ms"
        icon={Clock}
        color="rose"
        delay={4}
      />
    </div>
  )
}

export function UptimeCard({ metrics }: MetricsCardsProps) {
  return (
    <div className={cn(
      'rounded-xl border p-4 bg-gradient-to-br',
      'from-emerald-500/20 to-emerald-600/5 border-emerald-500/30',
      'animate-fade-in-up hover-lift'
    )}>
      <div className="flex items-center gap-2 mb-2">
        <Server className="w-4 h-4 text-emerald-400" />
        <span className="text-xs text-zinc-400">Uptime</span>
        <span className="ml-auto flex items-center gap-1">
          <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse-dot" />
          <span className="text-xs text-emerald-400">Online</span>
        </span>
      </div>
      <span className="text-2xl font-bold text-white">
        {formatDuration(metrics.uptime_seconds)}
      </span>
    </div>
  )
}
