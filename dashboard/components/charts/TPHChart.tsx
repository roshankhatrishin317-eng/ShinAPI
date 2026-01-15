'use client'

import React from 'react'
import { Clock } from 'lucide-react'
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip,
  ResponsiveContainer, Cell
} from 'recharts'
import { ChartCard } from './ChartCard'

interface TPHChartProps {
  data: Array<{
    timestamp: string
    tokens: number
    requests?: number
  }>
  currentTPH: number
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

function formatNumber(num: number): string {
  if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
  return Math.round(num).toString()
}

export function TPHChart({
  data,
  currentTPH,
  trend,
  expanded,
  onToggleExpand,
  onExport,
}: TPHChartProps) {
  const chartData = data.map((d, i) => {
    const date = d.timestamp ? new Date(d.timestamp) : new Date()
    date.setHours(date.getHours() - (data.length - i - 1))
    return {
      ...d,
      hour: date.toLocaleTimeString('en-US', {
        hour12: false,
        hour: '2-digit',
      }) + ':00',
      isPeak: false,
    }
  })

  // Mark peak hour
  if (chartData.length > 0) {
    const maxTokens = Math.max(...chartData.map(d => d.tokens))
    chartData.forEach(d => {
      if (d.tokens === maxTokens && maxTokens > 0) {
        d.isPeak = true
      }
    })
  }

  return (
    <ChartCard
      title="Tokens Per Hour"
      subtitle="Hourly token usage (24h)"
      icon={Clock}
      value={formatNumber(currentTPH)}
      suffix="TPH"
      trend={trend}
      color="amber"
      expanded={expanded}
      onToggleExpand={onToggleExpand}
      onExport={onExport}
    >
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={chartData} margin={{ top: 10, right: 10, bottom: 0, left: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" vertical={false} />
          <XAxis 
            dataKey="hour" 
            tick={{ fontSize: 9, fill: '#71717a' }} 
            axisLine={false}
            tickLine={false}
            interval={expanded ? 0 : 2}
          />
          <YAxis 
            tick={{ fontSize: 10, fill: '#71717a' }} 
            axisLine={false}
            tickLine={false}
            tickFormatter={formatNumber}
            width={45}
          />
          <Tooltip content={<CustomTooltip />} />
          <Bar
            dataKey="tokens"
            name="Tokens"
            radius={[4, 4, 0, 0]}
          >
            {chartData.map((entry, index) => (
              <Cell 
                key={`cell-${index}`} 
                fill={entry.isPeak ? '#f59e0b' : '#fbbf24'}
                fillOpacity={entry.isPeak ? 1 : 0.6}
              />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </ChartCard>
  )
}

export default TPHChart
