'use client'

import { useMemo } from 'react'
import { motion } from 'framer-motion'
import { cn } from '@/lib/utils'
import { 
  ResponsiveContainer, 
  PieChart, 
  Pie, 
  Cell, 
  Tooltip,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  ComposedChart,
  Line,
  Bar
} from 'recharts'

const COLORS = [
  '#3b82f6', '#8b5cf6', '#06b6d4', '#10b981', 
  '#f59e0b', '#ef4444', '#ec4899', '#6366f1'
]

interface ModelData {
  name: string
  requests: number
  tokens: number
}

interface ModelChartProps {
  data: ModelData[]
  maxItems?: number
}

export function ModelPieChart({ data, maxItems = 6 }: ModelChartProps) {
  const chartData = useMemo(() => 
    data.slice(0, maxItems).map((d, i) => ({
      ...d,
      color: COLORS[i % COLORS.length],
      displayName: d.name.length > 20 ? d.name.slice(0, 17) + '...' : d.name
    })),
  [data, maxItems])

  const totalRequests = chartData.reduce((sum, d) => sum + d.requests, 0)

  if (chartData.length === 0) {
    return (
      <div className="h-full flex items-center justify-center text-zinc-500 text-sm">
        No model data available
      </div>
    )
  }

  return (
    <div className="h-full flex flex-col">
      <div className="flex-1 min-h-[160px]">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={chartData}
              cx="50%"
              cy="50%"
              innerRadius="55%"
              outerRadius="85%"
              paddingAngle={3}
              dataKey="requests"
              animationBegin={0}
              animationDuration={800}
            >
              {chartData.map((entry, i) => (
                <Cell 
                  key={i} 
                  fill={entry.color}
                  stroke="transparent"
                  className="cursor-pointer hover:opacity-80 transition-opacity"
                />
              ))}
            </Pie>
            <Tooltip
              content={({ active, payload }) => {
                if (!active || !payload?.length) return null
                const data = payload[0].payload
                return (
                  <div className="bg-zinc-900/95 border border-white/10 rounded-lg px-3 py-2 shadow-xl">
                    <p className="text-sm font-medium text-white mb-1">{data.name}</p>
                    <p className="text-xs text-zinc-400">
                      {data.requests.toLocaleString()} requests
                    </p>
                    <p className="text-xs text-zinc-500">
                      {((data.requests / totalRequests) * 100).toFixed(1)}% of total
                    </p>
                  </div>
                )
              }}
            />
          </PieChart>
        </ResponsiveContainer>
      </div>

      <div className="mt-2 space-y-1.5 max-h-[100px] overflow-y-auto">
        {chartData.map((model, i) => (
          <motion.div
            key={model.name}
            initial={{ opacity: 0, x: -10 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: i * 0.05 }}
            className="flex items-center gap-2 text-xs group cursor-pointer"
          >
            <div 
              className="w-2.5 h-2.5 rounded-sm shrink-0 transition-transform group-hover:scale-125"
              style={{ backgroundColor: model.color }}
            />
            <span className="text-zinc-400 truncate flex-1 group-hover:text-white transition-colors">
              {model.displayName}
            </span>
            <span className="text-zinc-500 tabular-nums shrink-0">
              {model.requests.toLocaleString()}
            </span>
          </motion.div>
        ))}
      </div>
    </div>
  )
}

interface TimeSeriesData {
  time: string
  rpm: number
  tpm: number
  tps: number
}

interface RequestChartProps {
  data: TimeSeriesData[]
  height?: number
}

export function RequestChart({ data, height = 200 }: RequestChartProps) {
  if (data.length === 0) {
    return (
      <div className="flex items-center justify-center text-zinc-500 text-sm" style={{ height }}>
        Collecting data...
      </div>
    )
  }

  return (
    <ResponsiveContainer width="100%" height={height}>
      <ComposedChart data={data}>
        <defs>
          <linearGradient id="rpmGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#3b82f6" stopOpacity={0.4} />
            <stop offset="100%" stopColor="#3b82f6" stopOpacity={0} />
          </linearGradient>
          <linearGradient id="tpmGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#8b5cf6" stopOpacity={0.3} />
            <stop offset="100%" stopColor="#8b5cf6" stopOpacity={0} />
          </linearGradient>
        </defs>
        
        <XAxis 
          dataKey="time" 
          stroke="#52525b" 
          fontSize={10} 
          tickLine={false} 
          axisLine={false}
          interval="preserveStartEnd"
        />
        <YAxis 
          yAxisId="left"
          stroke="#52525b" 
          fontSize={10} 
          tickLine={false} 
          axisLine={false}
          width={35}
        />
        <YAxis 
          yAxisId="right"
          orientation="right"
          stroke="#52525b" 
          fontSize={10} 
          tickLine={false} 
          axisLine={false}
          width={35}
        />
        
        <Tooltip
          contentStyle={{
            backgroundColor: 'rgba(24,24,27,0.95)',
            border: '1px solid rgba(255,255,255,0.1)',
            borderRadius: '10px',
            padding: '10px 14px',
            boxShadow: '0 10px 40px rgba(0,0,0,0.5)'
          }}
          labelStyle={{ color: '#a1a1aa', marginBottom: '6px', fontSize: '11px' }}
          itemStyle={{ padding: '2px 0', fontSize: '12px' }}
        />
        
        <Area
          yAxisId="left"
          type="monotone"
          dataKey="rpm"
          stroke="#3b82f6"
          fill="url(#rpmGradient)"
          strokeWidth={2}
          name="RPM"
          dot={false}
          animationDuration={500}
        />
        
        <Line
          yAxisId="right"
          type="monotone"
          dataKey="tps"
          stroke="#06b6d4"
          strokeWidth={2}
          dot={false}
          name="TPS"
          animationDuration={500}
        />
      </ComposedChart>
    </ResponsiveContainer>
  )
}
