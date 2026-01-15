'use client'

import React from 'react'
import { Calendar } from 'lucide-react'
import {
  ComposedChart, Bar, Line, XAxis, YAxis, CartesianGrid, Tooltip,
  ResponsiveContainer, Legend
} from 'recharts'
import { ChartCard } from './ChartCard'

interface TPDChartProps {
  data: Array<{
    timestamp: string
    tokens: number
    requests?: number
  }>
  currentTPD: number
  trend?: number
  expanded?: boolean
  onToggleExpand?: () => void
  onExport?: () => void
  showRequests?: boolean
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

export function TPDChart({
  data,
  currentTPD,
  trend,
  expanded,
  onToggleExpand,
  onExport,
  showRequests = true,
}: TPDChartProps) {
  const chartData = data.map((d, i) => {
    const date = d.timestamp ? new Date(d.timestamp) : new Date()
    date.setDate(date.getDate() - (data.length - i - 1))
    return {
      ...d,
      day: date.toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
      }),
      dayShort: date.toLocaleDateString('en-US', {
        weekday: 'short',
      }),
    }
  })

  return (
    <ChartCard
      title="Tokens Per Day"
      subtitle="Daily token usage (30d)"
      icon={Calendar}
      value={formatNumber(currentTPD)}
      suffix="TPD"
      trend={trend}
      color="emerald"
      expanded={expanded}
      onToggleExpand={onToggleExpand}
      onExport={onExport}
    >
      <ResponsiveContainer width="100%" height="100%">
        <ComposedChart data={chartData} margin={{ top: 10, right: 10, bottom: 0, left: 0 }}>
          <defs>
            <linearGradient id="tpdGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#10b981" stopOpacity={0.8} />
              <stop offset="100%" stopColor="#10b981" stopOpacity={0.2} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" vertical={false} />
          <XAxis 
            dataKey="day" 
            tick={{ fontSize: 9, fill: '#71717a' }} 
            axisLine={false}
            tickLine={false}
            interval={expanded ? 0 : 4}
          />
          <YAxis 
            yAxisId="tokens"
            tick={{ fontSize: 10, fill: '#71717a' }} 
            axisLine={false}
            tickLine={false}
            tickFormatter={formatNumber}
            width={45}
          />
          {showRequests && (
            <YAxis 
              yAxisId="requests"
              orientation="right"
              tick={{ fontSize: 10, fill: '#71717a' }} 
              axisLine={false}
              tickLine={false}
              tickFormatter={formatNumber}
              width={35}
            />
          )}
          <Tooltip content={<CustomTooltip />} />
          <Bar
            yAxisId="tokens"
            dataKey="tokens"
            name="Tokens"
            fill="url(#tpdGradient)"
            radius={[4, 4, 0, 0]}
          />
          {showRequests && (
            <Line
              yAxisId="requests"
              type="monotone"
              dataKey="requests"
              name="Requests"
              stroke="#8b5cf6"
              strokeWidth={2}
              dot={false}
            />
          )}
        </ComposedChart>
      </ResponsiveContainer>
    </ChartCard>
  )
}

export default TPDChart
