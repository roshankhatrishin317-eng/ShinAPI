// Server Component - Charts rendered on server with CSS animations
// No Framer Motion, no heavy chart libraries on client

import { cn } from '@/lib/utils'
import { BarChart3, PieChart, Activity, TrendingUp } from 'lucide-react'

interface ModelStats {
  [model: string]: {
    requests: number
    tokens: number
  }
}

interface ChartsProps {
  modelStats?: ModelStats
  rpmHistory?: number[]
  tpmHistory?: number[]
}

// Simple SVG bar chart (server-rendered)
function SimpleBarChart({ 
  data, 
  height = 120, 
  color = '#3b82f6' 
}: { 
  data: number[]
  height?: number
  color?: string
}) {
  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-32 text-zinc-500 text-sm">
        No data available
      </div>
    )
  }

  const max = Math.max(...data, 1)
  const barWidth = 100 / data.length
  
  return (
    <svg 
      viewBox={`0 0 100 ${height}`} 
      className="w-full h-32"
      preserveAspectRatio="none"
    >
      {data.map((value, i) => {
        const barHeight = (value / max) * height * 0.9
        const y = height - barHeight
        return (
          <rect
            key={i}
            x={i * barWidth + barWidth * 0.1}
            y={y}
            width={barWidth * 0.8}
            height={barHeight}
            fill={color}
            opacity={0.8}
            rx={2}
            className="animate-bar-grow"
            style={{ 
              animationDelay: `${i * 30}ms`,
              transformOrigin: 'bottom'
            }}
          />
        )
      })}
    </svg>
  )
}

// Simple donut chart (server-rendered)
function SimpleDonutChart({ 
  data 
}: { 
  data: { name: string; value: number; color: string }[] 
}) {
  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-40 text-zinc-500 text-sm">
        No model data
      </div>
    )
  }

  const total = data.reduce((sum, item) => sum + item.value, 0)
  const radius = 40
  const circumference = 2 * Math.PI * radius
  let currentOffset = 0

  return (
    <div className="flex items-center gap-4">
      <svg viewBox="0 0 100 100" className="w-32 h-32">
        <circle
          cx="50"
          cy="50"
          r={radius}
          fill="none"
          stroke="#27272a"
          strokeWidth="12"
        />
        {data.map((item, i) => {
          const percentage = item.value / total
          const strokeDasharray = `${percentage * circumference} ${circumference}`
          const rotation = currentOffset * 360 - 90
          currentOffset += percentage
          
          return (
            <circle
              key={i}
              cx="50"
              cy="50"
              r={radius}
              fill="none"
              stroke={item.color}
              strokeWidth="12"
              strokeDasharray={strokeDasharray}
              strokeLinecap="round"
              transform={`rotate(${rotation} 50 50)`}
              className="animate-fade-in"
              style={{ animationDelay: `${i * 100}ms` }}
            />
          )
        })}
        <text
          x="50"
          y="50"
          textAnchor="middle"
          dominantBaseline="middle"
          className="fill-white text-xs font-bold"
        >
          {data.length}
        </text>
        <text
          x="50"
          y="58"
          textAnchor="middle"
          className="fill-zinc-500 text-[6px]"
        >
          models
        </text>
      </svg>
      
      {/* Legend */}
      <div className="flex-1 space-y-1">
        {data.slice(0, 5).map((item, i) => (
          <div 
            key={i} 
            className="flex items-center gap-2 text-xs animate-fade-in-right"
            style={{ animationDelay: `${i * 50}ms` }}
          >
            <span 
              className="w-2 h-2 rounded-full" 
              style={{ backgroundColor: item.color }}
            />
            <span className="text-zinc-400 truncate max-w-24">{item.name}</span>
            <span className="text-zinc-500 ml-auto">
              {((item.value / total) * 100).toFixed(0)}%
            </span>
          </div>
        ))}
        {data.length > 5 && (
          <div className="text-xs text-zinc-500">
            +{data.length - 5} more
          </div>
        )}
      </div>
    </div>
  )
}

// Gauge meter (server-rendered with CSS animation)
function GaugeMeter({ 
  value, 
  max, 
  label, 
  color = '#3b82f6' 
}: { 
  value: number
  max: number
  label: string
  color?: string
}) {
  const percentage = Math.min((value / max) * 100, 100)
  const radius = 35
  const circumference = Math.PI * radius // Half circle
  const offset = circumference - (circumference * percentage) / 100

  return (
    <div className="flex flex-col items-center">
      <svg viewBox="0 0 80 50" className="w-24 h-14">
        {/* Background arc */}
        <path
          d="M 5 45 A 35 35 0 0 1 75 45"
          fill="none"
          stroke="#27272a"
          strokeWidth="8"
          strokeLinecap="round"
        />
        {/* Animated arc */}
        <path
          d="M 5 45 A 35 35 0 0 1 75 45"
          fill="none"
          stroke={color}
          strokeWidth="8"
          strokeLinecap="round"
          strokeDasharray={circumference}
          strokeDashoffset={offset}
          className="animate-gauge-arc"
          style={{ 
            '--gauge-circumference': circumference,
            '--gauge-offset': offset
          } as React.CSSProperties}
        />
        {/* Value text */}
        <text
          x="40"
          y="42"
          textAnchor="middle"
          className="fill-white text-sm font-bold"
        >
          {value >= 1000 ? `${(value / 1000).toFixed(1)}K` : value.toFixed(0)}
        </text>
      </svg>
      <span className="text-xs text-zinc-400 mt-1">{label}</span>
    </div>
  )
}

// Card wrapper
function ChartCard({ 
  title, 
  icon: Icon, 
  color, 
  children,
  delay = 0
}: { 
  title: string
  icon: React.ElementType
  color: string
  children: React.ReactNode
  delay?: number
}) {
  const staggerClass = delay > 0 ? `stagger-${Math.min(delay, 8)}` : ''
  
  return (
    <div className={cn(
      'rounded-xl border border-zinc-800/50 bg-zinc-900/50 p-4',
      'animate-fade-in-up hover-lift',
      staggerClass
    )}>
      <div className="flex items-center gap-2 mb-4">
        <Icon className={cn('w-5 h-5', color)} />
        <span className="text-sm font-semibold text-zinc-300">{title}</span>
      </div>
      {children}
    </div>
  )
}

const MODEL_COLORS = [
  '#3b82f6', // blue
  '#8b5cf6', // violet
  '#06b6d4', // cyan
  '#10b981', // emerald
  '#f59e0b', // amber
  '#ef4444', // red
  '#ec4899', // pink
  '#6366f1', // indigo
]

export function ChartsSection({ modelStats, rpmHistory = [], tpmHistory = [] }: ChartsProps) {
  // Transform model stats for chart
  const modelData = modelStats 
    ? Object.entries(modelStats)
        .map(([name, stats], i) => ({
          name: name.split('/').pop() || name,
          value: stats.requests,
          color: MODEL_COLORS[i % MODEL_COLORS.length]
        }))
        .sort((a, b) => b.value - a.value)
    : []

  return (
    <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
      {/* Request History Chart */}
      <ChartCard 
        title="Request History" 
        icon={BarChart3} 
        color="text-blue-400"
        delay={1}
      >
        <SimpleBarChart 
          data={rpmHistory.slice(-30)} 
          color="#3b82f6" 
        />
      </ChartCard>

      {/* Token History Chart */}
      <ChartCard 
        title="Token History" 
        icon={Activity} 
        color="text-violet-400"
        delay={2}
      >
        <SimpleBarChart 
          data={tpmHistory.slice(-30)} 
          color="#8b5cf6" 
        />
      </ChartCard>

      {/* Model Distribution */}
      <ChartCard 
        title="Model Usage" 
        icon={PieChart} 
        color="text-cyan-400"
        delay={3}
      >
        <SimpleDonutChart data={modelData} />
      </ChartCard>
    </div>
  )
}

export function ThroughputGauges({ 
  tps = 0, 
  tpm = 0, 
  tph = 0, 
  tpd = 0 
}: { 
  tps?: number
  tpm?: number
  tph?: number
  tpd?: number
}) {
  return (
    <div className={cn(
      'rounded-xl border border-zinc-800/50 bg-zinc-900/50 p-6',
      'animate-fade-in-up'
    )}>
      <div className="flex items-center gap-3 mb-6">
        <div className="p-2 rounded-lg bg-gradient-to-br from-cyan-500/20 to-cyan-600/5 border border-cyan-500/30">
          <Activity className="w-5 h-5 text-cyan-400" />
        </div>
        <div>
          <h3 className="text-lg font-semibold text-white">Throughput Monitor</h3>
          <p className="text-xs text-zinc-500">Real-time request and token throughput</p>
        </div>
        <div className="ml-auto flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse-dot" />
          <span className="text-xs text-zinc-400">Live</span>
        </div>
      </div>

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <GaugeMeter value={tps} max={100} label="TPS" color="#06b6d4" />
        <GaugeMeter value={tpm} max={100000} label="TPM" color="#8b5cf6" />
        <GaugeMeter value={tph} max={1000000} label="TPH" color="#10b981" />
        <GaugeMeter value={tpd} max={10000000} label="TPD" color="#f59e0b" />
      </div>
    </div>
  )
}
