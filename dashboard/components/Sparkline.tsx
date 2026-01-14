'use client'

import { useMemo } from 'react'
import { motion } from 'framer-motion'
import { cn } from '@/lib/utils'

interface SparklineProps {
  data: number[]
  width?: number
  height?: number
  color?: string
  showArea?: boolean
  className?: string
}

export function Sparkline({ 
  data, 
  width = 80, 
  height = 24, 
  color = '#3b82f6',
  showArea = true,
  className 
}: SparklineProps) {
  const path = useMemo(() => {
    if (data.length < 2) return ''
    
    const max = Math.max(...data, 1)
    const min = Math.min(...data, 0)
    const range = max - min || 1
    
    const points = data.map((value, index) => {
      const x = (index / (data.length - 1)) * width
      const y = height - ((value - min) / range) * height
      return { x, y }
    })
    
    // Create smooth curve using quadratic bezier
    let d = `M ${points[0].x} ${points[0].y}`
    for (let i = 1; i < points.length; i++) {
      const prev = points[i - 1]
      const curr = points[i]
      const cpX = (prev.x + curr.x) / 2
      d += ` Q ${prev.x} ${prev.y} ${cpX} ${(prev.y + curr.y) / 2}`
    }
    d += ` L ${points[points.length - 1].x} ${points[points.length - 1].y}`
    
    return d
  }, [data, width, height])

  const areaPath = useMemo(() => {
    if (!showArea || data.length < 2) return ''
    return `${path} L ${width} ${height} L 0 ${height} Z`
  }, [path, showArea, data.length, width, height])

  if (data.length < 2) {
    return <div className={cn("opacity-30", className)} style={{ width, height }} />
  }

  return (
    <svg 
      width={width} 
      height={height} 
      className={cn("overflow-visible", className)}
      viewBox={`0 0 ${width} ${height}`}
    >
      <defs>
        <linearGradient id={`sparkline-gradient-${color}`} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={color} stopOpacity={0.3} />
          <stop offset="100%" stopColor={color} stopOpacity={0} />
        </linearGradient>
      </defs>
      
      {showArea && (
        <motion.path
          d={areaPath}
          fill={`url(#sparkline-gradient-${color})`}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.5 }}
        />
      )}
      
      <motion.path
        d={path}
        fill="none"
        stroke={color}
        strokeWidth={1.5}
        strokeLinecap="round"
        strokeLinejoin="round"
        initial={{ pathLength: 0, opacity: 0 }}
        animate={{ pathLength: 1, opacity: 1 }}
        transition={{ duration: 0.8, ease: "easeOut" }}
      />
      
      {/* Current value dot */}
      <motion.circle
        cx={width}
        cy={height - ((data[data.length - 1] - Math.min(...data)) / (Math.max(...data) - Math.min(...data) || 1)) * height}
        r={2}
        fill={color}
        initial={{ scale: 0 }}
        animate={{ scale: 1 }}
        transition={{ delay: 0.5 }}
      />
    </svg>
  )
}
