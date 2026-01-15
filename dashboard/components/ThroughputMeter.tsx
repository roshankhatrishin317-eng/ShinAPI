'use client'

import { motion } from 'framer-motion'
import { cn } from '@/lib/utils'
import { Activity, Zap, Server, TrendingUp } from 'lucide-react'

interface ThroughputMeterProps {
  tps: number
  tpm: number
  tph: number
  tpd: number
  maxTps?: number
  maxTpm?: number
  className?: string
}

function GaugeMeter({
  value,
  max,
  label,
  unit,
  icon: Icon,
  color,
  size = 'md'
}: {
  value: number
  max: number
  label: string
  unit: string
  icon: React.ElementType
  color: 'blue' | 'violet' | 'cyan' | 'emerald' | 'amber' | 'rose'
  size?: 'sm' | 'md' | 'lg'
}) {
  const percentage = Math.min((value / max) * 100, 100)
  
  const colorConfig = {
    blue: { stroke: '#3b82f6', glow: '#3b82f680', bg: 'from-blue-500/20 to-blue-600/5', text: 'text-blue-400' },
    violet: { stroke: '#8b5cf6', glow: '#8b5cf680', bg: 'from-violet-500/20 to-violet-600/5', text: 'text-violet-400' },
    cyan: { stroke: '#06b6d4', glow: '#06b6d480', bg: 'from-cyan-500/20 to-cyan-600/5', text: 'text-cyan-400' },
    emerald: { stroke: '#10b981', glow: '#10b98180', bg: 'from-emerald-500/20 to-emerald-600/5', text: 'text-emerald-400' },
    amber: { stroke: '#f59e0b', glow: '#f59e0b80', bg: 'from-amber-500/20 to-amber-600/5', text: 'text-amber-400' },
    rose: { stroke: '#f43f5e', glow: '#f43f5e80', bg: 'from-rose-500/20 to-rose-600/5', text: 'text-rose-400' },
  }

  const sizeConfig = {
    sm: { diameter: 80, stroke: 6, fontSize: 'text-lg', labelSize: 'text-[10px]' },
    md: { diameter: 100, stroke: 8, fontSize: 'text-xl', labelSize: 'text-xs' },
    lg: { diameter: 120, stroke: 10, fontSize: 'text-2xl', labelSize: 'text-sm' }
  }

  const { diameter, stroke: strokeWidth, fontSize, labelSize } = sizeConfig[size]
  const { stroke, glow, bg, text } = colorConfig[color]
  
  const radius = (diameter - strokeWidth) / 2
  const circumference = Math.PI * radius // Half circle (180 degrees)
  const strokeDashoffset = circumference - (circumference * percentage) / 100

  return (
    <motion.div
      className={cn(
        'relative flex flex-col items-center p-4 rounded-xl border border-zinc-800/50',
        'bg-gradient-to-br',
        bg
      )}
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      whileHover={{ scale: 1.02, y: -2 }}
      transition={{ type: 'spring', stiffness: 300, damping: 20 }}
    >
      <div className="flex items-center gap-2 mb-3">
        <Icon className={cn('w-4 h-4', text)} />
        <span className="text-xs font-medium text-zinc-400">{label}</span>
      </div>

      <div className="relative" style={{ width: diameter, height: diameter / 2 + 20 }}>
        <svg
          width={diameter}
          height={diameter / 2 + 20}
          className="overflow-visible"
        >
          {/* Background arc */}
          <path
            d={`M ${strokeWidth / 2} ${diameter / 2} A ${radius} ${radius} 0 0 1 ${diameter - strokeWidth / 2} ${diameter / 2}`}
            fill="none"
            stroke="currentColor"
            strokeWidth={strokeWidth}
            strokeLinecap="round"
            className="text-zinc-800"
          />
          
          {/* Animated progress arc */}
          <motion.path
            d={`M ${strokeWidth / 2} ${diameter / 2} A ${radius} ${radius} 0 0 1 ${diameter - strokeWidth / 2} ${diameter / 2}`}
            fill="none"
            stroke={stroke}
            strokeWidth={strokeWidth}
            strokeLinecap="round"
            strokeDasharray={circumference}
            initial={{ strokeDashoffset: circumference }}
            animate={{ strokeDashoffset }}
            transition={{ duration: 1, ease: 'easeOut' }}
            style={{
              filter: `drop-shadow(0 0 10px ${glow})`
            }}
          />

          {/* Tick marks */}
          {[0, 25, 50, 75, 100].map((tick) => {
            const angle = (tick / 100) * Math.PI
            const x1 = diameter / 2 - (radius - strokeWidth) * Math.cos(angle)
            const y1 = diameter / 2 - (radius - strokeWidth) * Math.sin(angle)
            const x2 = diameter / 2 - (radius - strokeWidth - 6) * Math.cos(angle)
            const y2 = diameter / 2 - (radius - strokeWidth - 6) * Math.sin(angle)
            return (
              <line
                key={tick}
                x1={x1}
                y1={y1}
                x2={x2}
                y2={y2}
                stroke="currentColor"
                strokeWidth={1.5}
                className="text-zinc-600"
              />
            )
          })}
        </svg>

        {/* Value display */}
        <div className="absolute inset-0 flex flex-col items-center justify-end pb-2">
          <motion.span
            className={cn('font-bold text-white', fontSize)}
            key={value}
            initial={{ opacity: 0, scale: 0.8 }}
            animate={{ opacity: 1, scale: 1 }}
          >
            {value >= 1000 ? `${(value / 1000).toFixed(1)}K` : value.toFixed(value < 10 ? 2 : 0)}
          </motion.span>
          <span className="text-[10px] text-zinc-500 uppercase tracking-wider">{unit}</span>
        </div>
      </div>

      {/* Percentage indicator */}
      <div className="flex items-center gap-1 mt-2">
        <div className="h-1 w-16 bg-zinc-800 rounded-full overflow-hidden">
          <motion.div
            className="h-full rounded-full"
            style={{ backgroundColor: stroke }}
            initial={{ width: 0 }}
            animate={{ width: `${percentage}%` }}
            transition={{ duration: 1, ease: 'easeOut' }}
          />
        </div>
        <span className={cn('text-[10px] font-medium', text)}>
          {percentage.toFixed(0)}%
        </span>
      </div>
    </motion.div>
  )
}

function LiveIndicator({ active }: { active: boolean }) {
  return (
    <div className="flex items-center gap-2">
      <motion.div
        className={cn(
          'w-2 h-2 rounded-full',
          active ? 'bg-emerald-500' : 'bg-zinc-600'
        )}
        animate={active ? {
          scale: [1, 1.2, 1],
          opacity: [1, 0.7, 1]
        } : {}}
        transition={{ duration: 1.5, repeat: Infinity }}
      />
      <span className="text-xs text-zinc-400">
        {active ? 'Live' : 'Idle'}
      </span>
    </div>
  )
}

export function ThroughputMeter({
  tps,
  tpm,
  tph,
  tpd,
  maxTps = 100,
  maxTpm = 100000,
  className
}: ThroughputMeterProps) {
  const isActive = tps > 0 || tpm > 0

  // Calculate estimated maximums based on current values
  const estimatedMaxTph = Math.max(tph * 1.5, 1000000)
  const estimatedMaxTpd = Math.max(tpd * 1.5, 10000000)

  return (
    <motion.div
      className={cn(
        'rounded-xl border border-zinc-800/50 bg-zinc-900/50 p-6',
        className
      )}
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
    >
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-gradient-to-br from-cyan-500/20 to-cyan-600/5 border border-cyan-500/30">
            <Activity className="w-5 h-5 text-cyan-400" />
          </div>
          <div>
            <h3 className="text-lg font-semibold text-white">Throughput Monitor</h3>
            <p className="text-xs text-zinc-500">Real-time request and token throughput</p>
          </div>
        </div>
        <LiveIndicator active={isActive} />
      </div>

      {/* Gauges Grid */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <GaugeMeter
          value={tps}
          max={maxTps}
          label="Requests/sec"
          unit="TPS"
          icon={Zap}
          color="cyan"
          size="md"
        />
        <GaugeMeter
          value={tpm}
          max={maxTpm}
          label="Tokens/min"
          unit="TPM"
          icon={Server}
          color="violet"
          size="md"
        />
        <GaugeMeter
          value={tph}
          max={estimatedMaxTph}
          label="Tokens/hour"
          unit="TPH"
          icon={TrendingUp}
          color="emerald"
          size="md"
        />
        <GaugeMeter
          value={tpd}
          max={estimatedMaxTpd}
          label="Tokens/day"
          unit="TPD"
          icon={Activity}
          color="amber"
          size="md"
        />
      </div>

      {/* Summary Stats */}
      <div className="mt-6 pt-4 border-t border-zinc-800/50">
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 text-center">
          <div>
            <p className="text-2xl font-bold text-white">
              {tps >= 1000 ? `${(tps / 1000).toFixed(1)}K` : tps.toFixed(2)}
            </p>
            <p className="text-xs text-zinc-500">Current TPS</p>
          </div>
          <div>
            <p className="text-2xl font-bold text-white">
              {tpm >= 1000000 ? `${(tpm / 1000000).toFixed(2)}M` : tpm >= 1000 ? `${(tpm / 1000).toFixed(1)}K` : tpm.toFixed(0)}
            </p>
            <p className="text-xs text-zinc-500">Current TPM</p>
          </div>
          <div>
            <p className="text-2xl font-bold text-white">
              {tph >= 1000000 ? `${(tph / 1000000).toFixed(2)}M` : tph >= 1000 ? `${(tph / 1000).toFixed(1)}K` : tph.toFixed(0)}
            </p>
            <p className="text-xs text-zinc-500">Hourly Tokens</p>
          </div>
          <div>
            <p className="text-2xl font-bold text-white">
              {tpd >= 1000000 ? `${(tpd / 1000000).toFixed(2)}M` : tpd >= 1000 ? `${(tpd / 1000).toFixed(1)}K` : tpd.toFixed(0)}
            </p>
            <p className="text-xs text-zinc-500">Daily Tokens</p>
          </div>
        </div>
      </div>
    </motion.div>
  )
}

export default ThroughputMeter
