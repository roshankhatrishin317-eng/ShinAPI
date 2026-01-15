'use client'

import React, { useState, useMemo } from 'react'
import { motion } from 'framer-motion'
import { 
  Maximize2, Minimize2, Settings, Download, 
  TrendingUp, TrendingDown, Minus
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { BorderBeam } from '@/components/ui/magic/border-beam'

interface ChartCardProps {
  title: string
  subtitle?: string
  icon?: React.ElementType
  value?: number | string
  suffix?: string
  trend?: number
  color?: 'blue' | 'violet' | 'cyan' | 'emerald' | 'amber' | 'rose'
  children: React.ReactNode
  expanded?: boolean
  onToggleExpand?: () => void
  onExport?: () => void
  showBeam?: boolean
  className?: string
}

const colorConfig = {
  blue: {
    gradient: 'from-blue-500/20 via-blue-500/10 to-transparent',
    border: 'border-blue-500/30',
    text: 'text-blue-400',
    beamFrom: '#3b82f6',
    beamTo: '#60a5fa',
  },
  violet: {
    gradient: 'from-violet-500/20 via-violet-500/10 to-transparent',
    border: 'border-violet-500/30',
    text: 'text-violet-400',
    beamFrom: '#8b5cf6',
    beamTo: '#a78bfa',
  },
  cyan: {
    gradient: 'from-cyan-500/20 via-cyan-500/10 to-transparent',
    border: 'border-cyan-500/30',
    text: 'text-cyan-400',
    beamFrom: '#06b6d4',
    beamTo: '#22d3ee',
  },
  emerald: {
    gradient: 'from-emerald-500/20 via-emerald-500/10 to-transparent',
    border: 'border-emerald-500/30',
    text: 'text-emerald-400',
    beamFrom: '#10b981',
    beamTo: '#34d399',
  },
  amber: {
    gradient: 'from-amber-500/20 via-amber-500/10 to-transparent',
    border: 'border-amber-500/30',
    text: 'text-amber-400',
    beamFrom: '#f59e0b',
    beamTo: '#fbbf24',
  },
  rose: {
    gradient: 'from-rose-500/20 via-rose-500/10 to-transparent',
    border: 'border-rose-500/30',
    text: 'text-rose-400',
    beamFrom: '#f43f5e',
    beamTo: '#fb7185',
  },
}

export function ChartCard({
  title,
  subtitle,
  icon: Icon,
  value,
  suffix,
  trend,
  color = 'blue',
  children,
  expanded = false,
  onToggleExpand,
  onExport,
  showBeam = true,
  className,
}: ChartCardProps) {
  const config = colorConfig[color]

  const TrendIcon = trend && trend > 0 ? TrendingUp : trend && trend < 0 ? TrendingDown : Minus
  const trendColor = trend && trend > 0 ? 'text-emerald-400' : trend && trend < 0 ? 'text-rose-400' : 'text-zinc-400'

  return (
    <motion.div
      layout
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className={cn(
        'relative overflow-hidden rounded-2xl border bg-zinc-900/80 backdrop-blur-xl',
        config.border,
        expanded ? 'col-span-full row-span-2' : '',
        className
      )}
    >
      {showBeam && (
        <BorderBeam
          size={expanded ? 400 : 200}
          duration={12}
          colorFrom={config.beamFrom}
          colorTo={config.beamTo}
        />
      )}

      <div className={cn('absolute inset-0 bg-gradient-to-br opacity-50', config.gradient)} />

      <div className="relative z-10 p-4">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-3">
            {Icon && (
              <div className={cn('p-2 rounded-xl bg-white/5', config.text)}>
                <Icon className="w-4 h-4" />
              </div>
            )}
            <div>
              <h3 className="text-sm font-semibold text-zinc-200">{title}</h3>
              {subtitle && <p className="text-xs text-zinc-500">{subtitle}</p>}
            </div>
          </div>

          <div className="flex items-center gap-2">
            {value !== undefined && (
              <div className="flex items-baseline gap-1 mr-2">
                <span className={cn('text-2xl font-bold', config.text)}>
                  {typeof value === 'number' ? value.toLocaleString() : value}
                </span>
                {suffix && <span className="text-sm text-zinc-500">{suffix}</span>}
                {trend !== undefined && (
                  <div className={cn('flex items-center gap-0.5 ml-2', trendColor)}>
                    <TrendIcon className="w-3 h-3" />
                    <span className="text-xs font-medium">
                      {trend > 0 ? '+' : ''}{trend.toFixed(1)}%
                    </span>
                  </div>
                )}
              </div>
            )}

            {onExport && (
              <button
                onClick={onExport}
                className="p-1.5 rounded-lg hover:bg-white/10 transition-colors text-zinc-400 hover:text-white"
              >
                <Download className="w-4 h-4" />
              </button>
            )}

            {onToggleExpand && (
              <button
                onClick={onToggleExpand}
                className="p-1.5 rounded-lg hover:bg-white/10 transition-colors text-zinc-400 hover:text-white"
              >
                {expanded ? <Minimize2 className="w-4 h-4" /> : <Maximize2 className="w-4 h-4" />}
              </button>
            )}
          </div>
        </div>

        <div className={cn('transition-all duration-300', expanded ? 'h-[400px]' : 'h-[200px]')}>
          {children}
        </div>
      </div>
    </motion.div>
  )
}

export default ChartCard
