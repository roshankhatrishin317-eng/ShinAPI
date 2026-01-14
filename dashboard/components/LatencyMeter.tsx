'use client'

import { motion } from 'framer-motion'
import { cn } from '@/lib/utils'

interface LatencyMeterProps {
  value: number
  label: string
  max?: number
  size?: 'sm' | 'md' | 'lg'
  showLabel?: boolean
}

function getColorFromValue(value: number, max: number): { 
  color: string
  bg: string
  glow: string
  status: string 
} {
  const percentage = (value / max) * 100

  if (percentage < 30) {
    return {
      color: '#10b981',
      bg: 'bg-emerald-500/20',
      glow: 'shadow-emerald-500/30',
      status: 'Excellent'
    }
  } else if (percentage < 60) {
    return {
      color: '#f59e0b',
      bg: 'bg-amber-500/20',
      glow: 'shadow-amber-500/30',
      status: 'Good'
    }
  } else {
    return {
      color: '#ef4444',
      bg: 'bg-rose-500/20',
      glow: 'shadow-rose-500/30',
      status: 'Slow'
    }
  }
}

export function LatencyMeter({ 
  value, 
  label, 
  max = 5000,
  size = 'md',
  showLabel = true 
}: LatencyMeterProps) {
  const percentage = Math.min((value / max) * 100, 100)
  const { color, bg, glow, status } = getColorFromValue(value, max)
  
  const sizeConfig = {
    sm: { diameter: 60, stroke: 4, fontSize: 'text-sm', labelSize: 'text-[9px]' },
    md: { diameter: 80, stroke: 5, fontSize: 'text-lg', labelSize: 'text-[10px]' },
    lg: { diameter: 100, stroke: 6, fontSize: 'text-xl', labelSize: 'text-xs' }
  }
  
  const { diameter, stroke, fontSize, labelSize } = sizeConfig[size]
  const radius = (diameter - stroke) / 2
  const circumference = 2 * Math.PI * radius

  return (
    <motion.div 
      className="flex flex-col items-center"
      initial={{ opacity: 0, scale: 0.8 }}
      animate={{ opacity: 1, scale: 1 }}
      whileHover={{ scale: 1.05 }}
    >
      <div className="relative" style={{ width: diameter, height: diameter }}>
        <svg
          width={diameter}
          height={diameter}
          className="-rotate-90"
        >
          <circle
            cx={diameter / 2}
            cy={diameter / 2}
            r={radius}
            fill="none"
            stroke="currentColor"
            strokeWidth={stroke}
            className="text-zinc-800"
          />
          
          <motion.circle
            cx={diameter / 2}
            cy={diameter / 2}
            r={radius}
            fill="none"
            stroke={color}
            strokeWidth={stroke}
            strokeLinecap="round"
            strokeDasharray={circumference}
            initial={{ strokeDashoffset: circumference }}
            animate={{ 
              strokeDashoffset: circumference - (circumference * percentage) / 100 
            }}
            transition={{ duration: 0.8, ease: "easeOut" }}
            style={{
              filter: `drop-shadow(0 0 8px ${color}40)`
            }}
          />
        </svg>

        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <motion.span 
            className={cn("font-bold text-white", fontSize)}
            key={value}
            initial={{ opacity: 0, y: 5 }}
            animate={{ opacity: 1, y: 0 }}
          >
            {Math.round(value)}
          </motion.span>
          <span className="text-[8px] text-zinc-500 uppercase tracking-wider">ms</span>
        </div>
      </div>

      {showLabel && (
        <p className={cn("mt-2 font-medium text-zinc-400", labelSize)}>{label}</p>
      )}
    </motion.div>
  )
}

interface LatencyGridProps {
  avg: number
  p50: number
  p95: number
  p99: number
}

export function LatencyGrid({ avg, p50, p95, p99 }: LatencyGridProps) {
  return (
    <div className="grid grid-cols-4 gap-4">
      <LatencyMeter value={avg} label="Average" size="md" />
      <LatencyMeter value={p50} label="P50" size="md" />
      <LatencyMeter value={p95} label="P95" size="md" />
      <LatencyMeter value={p99} label="P99" size="md" />
    </div>
  )
}
