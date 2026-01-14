'use client'

import React from 'react'
import {
  Card,
  Metric,
  Text,
  Flex,
  Grid,
  BadgeDelta,
  ProgressBar,
  ProgressCircle,
} from '@tremor/react'
import { motion } from 'framer-motion'
import { LucideIcon } from 'lucide-react'

interface MetricCardProps {
  title: string
  value: number | string
  icon?: LucideIcon
  trend?: number
  trendLabel?: string
  suffix?: string
  prefix?: string
  color?: 'blue' | 'violet' | 'cyan' | 'emerald' | 'amber' | 'rose'
  loading?: boolean
}

export function MetricCard({
  title,
  value,
  icon: Icon,
  trend,
  trendLabel,
  suffix = '',
  prefix = '',
  color = 'blue',
  loading = false,
}: MetricCardProps) {
  const colorMap = {
    blue: 'from-blue-500/20 to-blue-600/10 border-blue-500/30',
    violet: 'from-violet-500/20 to-violet-600/10 border-violet-500/30',
    cyan: 'from-cyan-500/20 to-cyan-600/10 border-cyan-500/30',
    emerald: 'from-emerald-500/20 to-emerald-600/10 border-emerald-500/30',
    amber: 'from-amber-500/20 to-amber-600/10 border-amber-500/30',
    rose: 'from-rose-500/20 to-rose-600/10 border-rose-500/30',
  }

  const iconColorMap = {
    blue: 'text-blue-400',
    violet: 'text-violet-400',
    cyan: 'text-cyan-400',
    emerald: 'text-emerald-400',
    amber: 'text-amber-400',
    rose: 'text-rose-400',
  }

  const getDeltaType = (t: number) => {
    if (t > 5) return 'increase'
    if (t > 0) return 'moderateIncrease'
    if (t < -5) return 'decrease'
    if (t < 0) return 'moderateDecrease'
    return 'unchanged'
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      whileHover={{ scale: 1.02 }}
      transition={{ duration: 0.2 }}
    >
      <Card className={`bg-gradient-to-br ${colorMap[color]} border ring-0 relative overflow-hidden`}>
        <Flex alignItems="start" justifyContent="between">
          <div className="truncate">
            <Text className="text-zinc-400 text-sm">{title}</Text>
            {loading ? (
              <div className="h-9 w-24 bg-zinc-700/50 animate-pulse rounded mt-1" />
            ) : (
              <Metric className="text-white text-3xl font-bold truncate">
                {prefix}{typeof value === 'number' ? value.toLocaleString() : value}{suffix}
              </Metric>
            )}
          </div>
          {Icon && (
            <div className={`p-3 rounded-xl bg-zinc-900/50 ${iconColorMap[color]}`}>
              <Icon className="w-5 h-5" />
            </div>
          )}
        </Flex>
        {trend !== undefined && (
          <Flex className="mt-4" justifyContent="start" alignItems="center">
            <BadgeDelta deltaType={getDeltaType(trend)} size="xs">
              {trend > 0 ? '+' : ''}{trend.toFixed(1)}%
            </BadgeDelta>
            {trendLabel && (
              <Text className="text-zinc-500 text-xs ml-2">{trendLabel}</Text>
            )}
          </Flex>
        )}
      </Card>
    </motion.div>
  )
}

export function CircularMetric({
  title,
  value,
  target,
  color = 'blue',
  size = 'md',
}: {
  title: string
  value: number
  target: number
  color?: 'blue' | 'violet' | 'cyan' | 'emerald' | 'amber' | 'rose'
  size?: 'sm' | 'md' | 'lg'
}) {
  const percentage = Math.min((value / target) * 100, 100)
  
  return (
    <Card className="bg-zinc-900/50 border-white/10 ring-0">
      <Flex flexDirection="col" alignItems="center">
        <ProgressCircle
          value={percentage}
          size={size}
          color={color}
          showAnimation
        >
          <span className="text-xl font-bold text-white">{Math.round(percentage)}%</span>
        </ProgressCircle>
        <Text className="text-zinc-400 mt-4">{title}</Text>
        <Text className="text-zinc-500 text-sm">
          {value.toLocaleString()} / {target.toLocaleString()}
        </Text>
      </Flex>
    </Card>
  )
}

export function MetricsGrid({
  children,
  cols = 4,
}: {
  children: React.ReactNode
  cols?: 1 | 2 | 3 | 4 | 5 | 6
}) {
  return (
    <Grid numItemsSm={2} numItemsMd={cols} className="gap-4">
      {children}
    </Grid>
  )
}
