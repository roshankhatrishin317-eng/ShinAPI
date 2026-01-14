'use client'

import React, { useRef, useEffect, useState } from 'react'
import { motion, useSpring, useTransform, AnimatePresence } from 'framer-motion'
import { 
  TrendingUp, TrendingDown, Minus, Activity, Zap, 
  Clock, Database, ArrowUpRight, ArrowDownRight,
  Cpu, Gauge, BarChart3
} from 'lucide-react'
import { 
  AreaChart, Area, XAxis, YAxis, ResponsiveContainer, 
  Tooltip, CartesianGrid, ReferenceLine 
} from 'recharts'
import { cn } from '@/lib/utils'

interface LiveMetricCardProps {
  title: string
  value: number
  previousValue?: number
  suffix?: string
  prefix?: string
  icon: React.ElementType
  color: 'blue' | 'violet' | 'cyan' | 'emerald' | 'amber' | 'rose'
  sparklineData: number[]
  trend?: number
  format?: 'number' | 'decimal' | 'percentage'
  showGraph?: boolean
  maxDataPoints?: number
}

const colorConfig = {
  blue: {
    gradient: 'from-blue-500/20 via-blue-500/10 to-transparent',
    border: 'border-blue-500/30',
    text: 'text-blue-400',
    fill: '#3b82f6',
    fillOpacity: 0.3,
    stroke: '#3b82f6',
    glow: 'shadow-blue-500/20',
    bg: 'bg-blue-500/10',
  },
  violet: {
    gradient: 'from-violet-500/20 via-violet-500/10 to-transparent',
    border: 'border-violet-500/30',
    text: 'text-violet-400',
    fill: '#8b5cf6',
    fillOpacity: 0.3,
    stroke: '#8b5cf6',
    glow: 'shadow-violet-500/20',
    bg: 'bg-violet-500/10',
  },
  cyan: {
    gradient: 'from-cyan-500/20 via-cyan-500/10 to-transparent',
    border: 'border-cyan-500/30',
    text: 'text-cyan-400',
    fill: '#06b6d4',
    fillOpacity: 0.3,
    stroke: '#06b6d4',
    glow: 'shadow-cyan-500/20',
    bg: 'bg-cyan-500/10',
  },
  emerald: {
    gradient: 'from-emerald-500/20 via-emerald-500/10 to-transparent',
    border: 'border-emerald-500/30',
    text: 'text-emerald-400',
    fill: '#10b981',
    fillOpacity: 0.3,
    stroke: '#10b981',
    glow: 'shadow-emerald-500/20',
    bg: 'bg-emerald-500/10',
  },
  amber: {
    gradient: 'from-amber-500/20 via-amber-500/10 to-transparent',
    border: 'border-amber-500/30',
    text: 'text-amber-400',
    fill: '#f59e0b',
    fillOpacity: 0.3,
    stroke: '#f59e0b',
    glow: 'shadow-amber-500/20',
    bg: 'bg-amber-500/10',
  },
  rose: {
    gradient: 'from-rose-500/20 via-rose-500/10 to-transparent',
    border: 'border-rose-500/30',
    text: 'text-rose-400',
    fill: '#f43f5e',
    fillOpacity: 0.3,
    stroke: '#f43f5e',
    glow: 'shadow-rose-500/20',
    bg: 'bg-rose-500/10',
  },
}

// Animated number that smoothly transitions
function AnimatedNumber({ 
  value, 
  format = 'number',
  suffix = '',
  prefix = ''
}: { 
  value: number
  format?: 'number' | 'decimal' | 'percentage'
  suffix?: string
  prefix?: string
}) {
  const spring = useSpring(0, { stiffness: 100, damping: 30 })
  const display = useTransform(spring, (current) => {
    if (format === 'decimal') return current.toFixed(1)
    if (format === 'percentage') return current.toFixed(1)
    return Math.round(current).toLocaleString()
  })
  
  useEffect(() => {
    spring.set(value)
  }, [value, spring])

  return (
    <motion.span>
      {prefix}
      <motion.span>{display}</motion.span>
      {suffix}
    </motion.span>
  )
}

// Mini sparkline for inline display
function MiniSparkline({ 
  data, 
  color, 
  width = 60, 
  height = 24 
}: { 
  data: number[]
  color: string
  width?: number
  height?: number
}) {
  if (data.length < 2) return null
  
  const max = Math.max(...data, 1)
  const min = Math.min(...data, 0)
  const range = max - min || 1
  
  const points = data.map((v, i) => {
    const x = (i / (data.length - 1)) * width
    const y = height - ((v - min) / range) * height
    return `${x},${y}`
  }).join(' ')

  return (
    <svg width={width} height={height} className="overflow-visible">
      <defs>
        <linearGradient id={`spark-${color}`} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={color} stopOpacity="0.5" />
          <stop offset="100%" stopColor={color} stopOpacity="0" />
        </linearGradient>
      </defs>
      <polyline
        points={points}
        fill="none"
        stroke={color}
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

// Trend indicator with animation
function TrendIndicator({ trend, color }: { trend: number; color: string }) {
  const isPositive = trend > 0
  const isNeutral = trend === 0
  
  return (
    <motion.div
      initial={{ opacity: 0, y: 5 }}
      animate={{ opacity: 1, y: 0 }}
      className={cn(
        'flex items-center gap-1 text-xs font-medium px-2 py-1 rounded-full',
        isPositive ? 'bg-emerald-500/10 text-emerald-400' :
        isNeutral ? 'bg-zinc-500/10 text-zinc-400' :
        'bg-rose-500/10 text-rose-400'
      )}
    >
      {isPositive ? (
        <ArrowUpRight className="w-3 h-3" />
      ) : isNeutral ? (
        <Minus className="w-3 h-3" />
      ) : (
        <ArrowDownRight className="w-3 h-3" />
      )}
      <span>{isPositive ? '+' : ''}{trend.toFixed(1)}%</span>
    </motion.div>
  )
}

// Pulsing live indicator
function LiveIndicator() {
  return (
    <div className="flex items-center gap-1.5">
      <span className="relative flex h-2 w-2">
        <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
        <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500" />
      </span>
      <span className="text-[10px] font-medium text-green-400 uppercase tracking-wider">Live</span>
    </div>
  )
}

export function LiveMetricCard({
  title,
  value,
  previousValue,
  suffix = '',
  prefix = '',
  icon: Icon,
  color,
  sparklineData,
  trend,
  format = 'number',
  showGraph = true,
  maxDataPoints = 20,
}: LiveMetricCardProps) {
  const config = colorConfig[color]
  const chartData = sparklineData.slice(-maxDataPoints).map((v, i) => ({ 
    index: i, 
    value: v,
    time: `${i}s ago`
  }))
  
  const calculatedTrend = trend ?? (previousValue 
    ? ((value - previousValue) / Math.max(previousValue, 1)) * 100 
    : 0)

  return (
    <motion.div
      initial={{ opacity: 0, y: 20, scale: 0.95 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      whileHover={{ scale: 1.02, y: -2 }}
      transition={{ duration: 0.3 }}
      className={cn(
        'relative overflow-hidden rounded-2xl border p-5',
        'bg-gradient-to-br from-zinc-900/90 to-zinc-900/50',
        'backdrop-blur-xl shadow-2xl',
        config.border,
        config.glow
      )}
    >
      {/* Background gradient effect */}
      <div className={cn(
        'absolute inset-0 bg-gradient-to-br opacity-50',
        config.gradient
      )} />
      
      {/* Animated background particles */}
      <div className="absolute inset-0 overflow-hidden">
        {[...Array(3)].map((_, i) => (
          <motion.div
            key={i}
            className={cn('absolute w-32 h-32 rounded-full blur-3xl', config.bg)}
            animate={{
              x: [0, 50, 0],
              y: [0, 30, 0],
              scale: [1, 1.2, 1],
            }}
            transition={{
              duration: 8 + i * 2,
              repeat: Infinity,
              ease: 'easeInOut',
              delay: i * 2,
            }}
            style={{
              left: `${20 + i * 30}%`,
              top: `${10 + i * 20}%`,
            }}
          />
        ))}
      </div>

      <div className="relative z-10">
        {/* Header */}
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <div className={cn('p-2 rounded-xl', config.bg)}>
              <Icon className={cn('w-4 h-4', config.text)} />
            </div>
            <span className="text-sm font-medium text-zinc-400">{title}</span>
          </div>
          <LiveIndicator />
        </div>

        {/* Value */}
        <div className="flex items-end justify-between mb-4">
          <div className="flex items-baseline gap-1">
            <span className="text-4xl font-bold text-white tracking-tight">
              <AnimatedNumber value={value} format={format} prefix={prefix} />
            </span>
            {suffix && (
              <span className="text-lg text-zinc-500 font-medium">{suffix}</span>
            )}
          </div>
          <TrendIndicator trend={calculatedTrend} color={config.fill} />
        </div>

        {/* Real-time Graph */}
        {showGraph && chartData.length > 1 && (
          <div className="h-24 -mx-2 mt-2">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData} margin={{ top: 5, right: 5, bottom: 0, left: 5 }}>
                <defs>
                  <linearGradient id={`gradient-${color}`} x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor={config.fill} stopOpacity={0.4} />
                    <stop offset="50%" stopColor={config.fill} stopOpacity={0.1} />
                    <stop offset="100%" stopColor={config.fill} stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid 
                  strokeDasharray="3 3" 
                  stroke="rgba(255,255,255,0.05)" 
                  vertical={false}
                />
                <XAxis 
                  dataKey="index" 
                  hide 
                />
                <YAxis 
                  hide 
                  domain={['dataMin - 1', 'dataMax + 1']}
                />
                <Tooltip
                  content={({ active, payload }) => {
                    if (active && payload?.[0]) {
                      return (
                        <div className="bg-zinc-900 border border-white/10 rounded-lg px-3 py-2 shadow-xl">
                          <span className={cn('text-sm font-medium', config.text)}>
                            {prefix}{payload[0].value?.toLocaleString()}{suffix}
                          </span>
                        </div>
                      )
                    }
                    return null
                  }}
                />
                <Area
                  type="monotone"
                  dataKey="value"
                  stroke={config.stroke}
                  strokeWidth={2}
                  fill={`url(#gradient-${color})`}
                  animationDuration={300}
                />
                {/* Current value reference line */}
                <ReferenceLine
                  y={value}
                  stroke={config.stroke}
                  strokeDasharray="3 3"
                  strokeOpacity={0.5}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        )}

        {/* Footer stats */}
        <div className="flex items-center justify-between mt-3 pt-3 border-t border-white/5">
          <div className="flex items-center gap-4">
            <div className="text-xs text-zinc-500">
              <span className="text-zinc-400">Max:</span>{' '}
              <span className={config.text}>{Math.max(...sparklineData, 0).toLocaleString()}</span>
            </div>
            <div className="text-xs text-zinc-500">
              <span className="text-zinc-400">Avg:</span>{' '}
              <span className="text-zinc-300">
                {sparklineData.length > 0 
                  ? Math.round(sparklineData.reduce((a, b) => a + b, 0) / sparklineData.length).toLocaleString()
                  : 0}
              </span>
            </div>
          </div>
          <MiniSparkline 
            data={sparklineData.slice(-10)} 
            color={config.stroke} 
          />
        </div>
      </div>
    </motion.div>
  )
}

// Grid layout for live metrics
export function LiveMetricsGrid({ children }: { children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      {children}
    </div>
  )
}

// Compact stat card for secondary metrics
export function CompactStatCard({
  title,
  value,
  icon: Icon,
  color,
  trend,
  suffix = '',
}: {
  title: string
  value: number | string
  icon: React.ElementType
  color: 'blue' | 'violet' | 'cyan' | 'emerald' | 'amber' | 'rose'
  trend?: number
  suffix?: string
}) {
  const config = colorConfig[color]
  
  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      whileHover={{ scale: 1.02 }}
      className={cn(
        'relative overflow-hidden rounded-xl border p-4',
        'bg-zinc-900/50 backdrop-blur-sm',
        config.border
      )}
    >
      <div className={cn(
        'absolute top-0 right-0 w-20 h-20 -mr-10 -mt-10 rounded-full blur-2xl opacity-30',
        config.bg
      )} />
      
      <div className="relative z-10 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className={cn('p-2 rounded-lg', config.bg)}>
            <Icon className={cn('w-4 h-4', config.text)} />
          </div>
          <div>
            <p className="text-xs text-zinc-500">{title}</p>
            <p className="text-lg font-bold text-white">
              {typeof value === 'number' ? value.toLocaleString() : value}{suffix}
            </p>
          </div>
        </div>
        {trend !== undefined && (
          <div className={cn(
            'text-xs font-medium',
            trend > 0 ? 'text-emerald-400' : trend < 0 ? 'text-rose-400' : 'text-zinc-400'
          )}>
            {trend > 0 ? '+' : ''}{trend.toFixed(1)}%
          </div>
        )}
      </div>
    </motion.div>
  )
}
