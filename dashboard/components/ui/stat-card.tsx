'use client'

import { useEffect, useRef, useState } from 'react'
import { motion } from 'framer-motion'
import { cn } from '@/lib/utils'

interface StatCardProps {
  title: string
  value: number | string
  suffix?: string
  prefix?: string
  icon: React.ElementType
  trend?: number
  trendLabel?: string
  color: 'blue' | 'purple' | 'cyan' | 'emerald' | 'amber' | 'rose'
  sparklineData?: number[]
  animate?: boolean
  size?: 'sm' | 'md' | 'lg'
}

const colorConfig = {
  blue: {
    gradient: 'from-blue-500 to-blue-600',
    glow: 'shadow-blue-500/25',
    text: 'text-blue-400',
    bg: 'bg-blue-500/10',
    border: 'border-blue-500/20',
    ring: '#3b82f6'
  },
  purple: {
    gradient: 'from-purple-500 to-purple-600',
    glow: 'shadow-purple-500/25',
    text: 'text-purple-400',
    bg: 'bg-purple-500/10',
    border: 'border-purple-500/20',
    ring: '#a855f7'
  },
  cyan: {
    gradient: 'from-cyan-500 to-cyan-600',
    glow: 'shadow-cyan-500/25',
    text: 'text-cyan-400',
    bg: 'bg-cyan-500/10',
    border: 'border-cyan-500/20',
    ring: '#06b6d4'
  },
  emerald: {
    gradient: 'from-emerald-500 to-emerald-600',
    glow: 'shadow-emerald-500/25',
    text: 'text-emerald-400',
    bg: 'bg-emerald-500/10',
    border: 'border-emerald-500/20',
    ring: '#10b981'
  },
  amber: {
    gradient: 'from-amber-500 to-amber-600',
    glow: 'shadow-amber-500/25',
    text: 'text-amber-400',
    bg: 'bg-amber-500/10',
    border: 'border-amber-500/20',
    ring: '#f59e0b'
  },
  rose: {
    gradient: 'from-rose-500 to-rose-600',
    glow: 'shadow-rose-500/25',
    text: 'text-rose-400',
    bg: 'bg-rose-500/10',
    border: 'border-rose-500/20',
    ring: '#f43f5e'
  }
}

function AnimatedNumber({ value, decimals = 0 }: { value: number; decimals?: number }) {
  const [display, setDisplay] = useState(value)
  const prevValue = useRef(value)

  useEffect(() => {
    const start = prevValue.current
    const diff = value - start
    const duration = 400
    const startTime = Date.now()

    const animate = () => {
      const elapsed = Date.now() - startTime
      const progress = Math.min(elapsed / duration, 1)
      const eased = 1 - Math.pow(1 - progress, 4)
      setDisplay(start + diff * eased)
      if (progress < 1) requestAnimationFrame(animate)
      else prevValue.current = value
    }

    requestAnimationFrame(animate)
  }, [value])

  const formatValue = (num: number) => {
    if (decimals > 0) return num.toFixed(decimals)
    if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
    if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
    return Math.round(num).toLocaleString()
  }

  return <>{formatValue(display)}</>
}

function MiniSparkline({ data, color, width = 64, height = 28 }: { 
  data: number[], 
  color: string,
  width?: number,
  height?: number 
}) {
  if (data.length < 2) return null

  const max = Math.max(...data, 1)
  const min = Math.min(...data, 0)
  const range = max - min || 1

  const points = data.map((v, i) => ({
    x: (i / (data.length - 1)) * width,
    y: height - ((v - min) / range) * (height - 4) - 2
  }))

  const linePath = points.map((p, i) => 
    i === 0 ? `M ${p.x} ${p.y}` : `L ${p.x} ${p.y}`
  ).join(' ')

  const areaPath = `${linePath} L ${width} ${height} L 0 ${height} Z`

  return (
    <svg width={width} height={height} className="overflow-visible">
      <defs>
        <linearGradient id={`spark-${color}`} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={color} stopOpacity={0.3} />
          <stop offset="100%" stopColor={color} stopOpacity={0} />
        </linearGradient>
      </defs>
      <motion.path
        d={areaPath}
        fill={`url(#spark-${color})`}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
      />
      <motion.path
        d={linePath}
        fill="none"
        stroke={color}
        strokeWidth={1.5}
        strokeLinecap="round"
        initial={{ pathLength: 0 }}
        animate={{ pathLength: 1 }}
        transition={{ duration: 0.6 }}
      />
      <motion.circle
        cx={width}
        cy={points[points.length - 1]?.y || height / 2}
        r={3}
        fill={color}
        initial={{ scale: 0 }}
        animate={{ scale: 1 }}
        transition={{ delay: 0.5 }}
      />
    </svg>
  )
}

export function StatCard({
  title,
  value,
  suffix,
  prefix,
  icon: Icon,
  trend,
  trendLabel,
  color,
  sparklineData,
  animate = true,
  size = 'md'
}: StatCardProps) {
  const config = colorConfig[color]
  const numericValue = typeof value === 'number' ? value : parseFloat(value) || 0

  return (
    <motion.div
      initial={{ opacity: 0, y: 20, scale: 0.95 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      whileHover={{ y: -6, transition: { duration: 0.2 } }}
      className={cn(
        "relative overflow-hidden rounded-2xl",
        "bg-zinc-900/50 backdrop-blur-xl",
        "border border-white/[0.08]",
        "hover:border-white/[0.15] hover:bg-zinc-900/70",
        "transition-all duration-300",
        "group cursor-default",
        size === 'sm' && "p-4",
        size === 'md' && "p-5",
        size === 'lg' && "p-6"
      )}
    >
      <div className={cn(
        "absolute top-0 left-0 right-0 h-[2px]",
        "bg-gradient-to-r",
        config.gradient,
        "opacity-80"
      )} />

      <div className={cn(
        "absolute -top-20 -right-20 w-40 h-40 rounded-full blur-3xl",
        "transition-opacity duration-500 opacity-0 group-hover:opacity-100",
        `bg-gradient-to-br ${config.gradient}`
      )} style={{ opacity: 0.15 }} />

      <div className="relative z-10">
        <div className="flex items-start justify-between mb-4">
          <motion.div
            whileHover={{ rotate: 10, scale: 1.1 }}
            className={cn(
              "p-2.5 rounded-xl",
              "bg-gradient-to-br shadow-lg",
              config.gradient,
              config.glow
            )}
          >
            <Icon className="w-5 h-5 text-white" />
          </motion.div>

          {trend !== undefined && trend !== 0 && (
            <motion.div
              initial={{ opacity: 0, x: 10 }}
              animate={{ opacity: 1, x: 0 }}
              className={cn(
                "flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium",
                trend > 0 ? "bg-emerald-500/15 text-emerald-400" : "bg-rose-500/15 text-rose-400"
              )}
            >
              <svg 
                className={cn("w-3 h-3", trend < 0 && "rotate-180")} 
                fill="none" 
                viewBox="0 0 24 24" 
                stroke="currentColor"
              >
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 10l7-7m0 0l7 7m-7-7v18" />
              </svg>
              {Math.abs(trend).toFixed(1)}%
            </motion.div>
          )}
        </div>

        <div className="flex items-end justify-between">
          <div>
            <div className="flex items-baseline gap-1">
              {prefix && <span className="text-lg text-zinc-500">{prefix}</span>}
              <span className="text-3xl font-bold text-white tracking-tight">
                {animate ? <AnimatedNumber value={numericValue} decimals={suffix === '%' ? 1 : 0} /> : value}
              </span>
              {suffix && <span className="text-lg text-zinc-500 ml-0.5">{suffix}</span>}
            </div>
            <p className="text-sm text-zinc-500 mt-1 font-medium">{title}</p>
            {trendLabel && (
              <p className="text-xs text-zinc-600 mt-0.5">{trendLabel}</p>
            )}
          </div>

          {sparklineData && sparklineData.length > 1 && (
            <MiniSparkline 
              data={sparklineData} 
              color={config.ring}
              width={70}
              height={32}
            />
          )}
        </div>
      </div>
    </motion.div>
  )
}
