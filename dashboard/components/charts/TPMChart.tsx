'use client'

import React from 'react'
import { Database } from 'lucide-react'
import {
  AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip,
  ResponsiveContainer, Legend
} from 'recharts'
import { ChartCard } from './ChartCard'

interface TPMChartProps {
  data: Array<{
    timestamp: string
    tokens: number
    input_tokens?: number
    output_tokens?: number
  }>
  currentTPM: number
  trend?: number
  expanded?: boolean
  onToggleExpand?: () => void
  onExport?: () => void
  showBreakdown?: boolean
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

export function TPMChart({
  data,
  currentTPM,
  trend,
  expanded,
  onToggleExpand,
  onExport,
  showBreakdown = true,
}: TPMChartProps) {
  const chartData = data.map((d, i) => ({
    ...d,
    time: d.timestamp ? new Date(d.timestamp).toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
    }) : `${data.length - i}m ago`,
    input: d.input_tokens || Math.round(d.tokens * 0.4),
    output: d.output_tokens || Math.round(d.tokens * 0.6),
  }))

  return (
    <ChartCard
      title="Tokens Per Minute"
      subtitle="Token throughput over time"
      icon={Database}
      value={formatNumber(currentTPM)}
      suffix="TPM"
      trend={trend}
      color="violet"
      expanded={expanded}
      onToggleExpand={onToggleExpand}
      onExport={onExport}
    >
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={chartData} margin={{ top: 10, right: 10, bottom: 0, left: 0 }}>
          <defs>
            <linearGradient id="inputGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#3b82f6" stopOpacity={0.4} />
              <stop offset="100%" stopColor="#3b82f6" stopOpacity={0} />
            </linearGradient>
            <linearGradient id="outputGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#10b981" stopOpacity={0.4} />
              <stop offset="100%" stopColor="#10b981" stopOpacity={0} />
            </linearGradient>
            <linearGradient id="tokensGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#8b5cf6" stopOpacity={0.4} />
              <stop offset="100%" stopColor="#8b5cf6" stopOpacity={0} />
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
            tickFormatter={formatNumber}
            width={45}
          />
          <Tooltip content={<CustomTooltip />} />
          {showBreakdown ? (
            <>
              <Area
                type="monotone"
                dataKey="input"
                name="Input Tokens"
                stackId="1"
                stroke="#3b82f6"
                fill="url(#inputGradient)"
                strokeWidth={1.5}
              />
              <Area
                type="monotone"
                dataKey="output"
                name="Output Tokens"
                stackId="1"
                stroke="#10b981"
                fill="url(#outputGradient)"
                strokeWidth={1.5}
              />
            </>
          ) : (
            <Area
              type="monotone"
              dataKey="tokens"
              name="Tokens"
              stroke="#8b5cf6"
              fill="url(#tokensGradient)"
              strokeWidth={2}
            />
          )}
        </AreaChart>
      </ResponsiveContainer>
    </ChartCard>
  )
}

export default TPMChart
