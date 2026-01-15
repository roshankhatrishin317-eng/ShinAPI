'use client'

import React from 'react'
import { Zap } from 'lucide-react'
import {
  AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip,
  ResponsiveContainer, ReferenceLine
} from 'recharts'
import { ChartCard } from './ChartCard'

interface TPSChartProps {
  data: Array<{
    timestamp: string
    requests: number
    tokens?: number
  }>
  currentTPS: number
  trend?: number
  expanded?: boolean
  onToggleExpand?: () => void
  onExport?: () => void
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
            {entry.value?.toLocaleString()}
          </span>
        </div>
      ))}
    </div>
  )
}

export function TPSChart({
  data,
  currentTPS,
  trend,
  expanded,
  onToggleExpand,
  onExport,
}: TPSChartProps) {
  const chartData = data.map((d, i) => ({
    ...d,
    time: d.timestamp ? new Date(d.timestamp).toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    }) : `${data.length - i}s ago`,
  }))

  const maxValue = Math.max(...data.map(d => d.requests), 1)
  const avgValue = data.length > 0 
    ? data.reduce((sum, d) => sum + d.requests, 0) / data.length 
    : 0

  return (
    <ChartCard
      title="Transactions Per Second"
      subtitle="Real-time request throughput"
      icon={Zap}
      value={currentTPS.toFixed(1)}
      suffix="TPS"
      trend={trend}
      color="cyan"
      expanded={expanded}
      onToggleExpand={onToggleExpand}
      onExport={onExport}
    >
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={chartData} margin={{ top: 10, right: 10, bottom: 0, left: 0 }}>
          <defs>
            <linearGradient id="tpsGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#06b6d4" stopOpacity={0.4} />
              <stop offset="50%" stopColor="#06b6d4" stopOpacity={0.1} />
              <stop offset="100%" stopColor="#06b6d4" stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" />
          <XAxis 
            dataKey="time" 
            tick={{ fontSize: 10, fill: '#71717a' }} 
            axisLine={false}
            tickLine={false}
          />
          <YAxis 
            tick={{ fontSize: 10, fill: '#71717a' }} 
            axisLine={false}
            tickLine={false}
            domain={[0, Math.ceil(maxValue * 1.2)]}
            width={35}
          />
          <Tooltip content={<CustomTooltip />} />
          <ReferenceLine 
            y={avgValue} 
            stroke="#06b6d4" 
            strokeDasharray="3 3" 
            strokeOpacity={0.5}
          />
          <Area
            type="monotone"
            dataKey="requests"
            name="TPS"
            stroke="#06b6d4"
            fill="url(#tpsGradient)"
            strokeWidth={2}
            dot={false}
            animationDuration={300}
          />
        </AreaChart>
      </ResponsiveContainer>
    </ChartCard>
  )
}

export default TPSChart
