'use client'

import { motion, AnimatePresence } from 'framer-motion'
import { cn } from '@/lib/utils'
import { 
  CheckCircle2, XCircle, Clock, Zap, 
  ArrowRight, ChevronRight
} from 'lucide-react'

interface ActivityItem {
  id: string
  timestamp: number
  model: string
  tokens: number
  latency_ms: number
  status: 'success' | 'error' | 'rate_limited'
  auth_id: string
  endpoint?: string
}

interface ActivityFeedProps {
  items: ActivityItem[]
  maxItems?: number
}

const statusConfig = {
  success: {
    icon: CheckCircle2,
    color: 'text-emerald-400',
    bg: 'bg-emerald-500/10',
    border: 'border-emerald-500/20',
    glow: 'shadow-emerald-500/20'
  },
  error: {
    icon: XCircle,
    color: 'text-rose-400',
    bg: 'bg-rose-500/10',
    border: 'border-rose-500/20',
    glow: 'shadow-rose-500/20'
  },
  rate_limited: {
    icon: Clock,
    color: 'text-amber-400',
    bg: 'bg-amber-500/10',
    border: 'border-amber-500/20',
    glow: 'shadow-amber-500/20'
  }
}

function getLatencyColor(ms: number): string {
  if (ms < 500) return 'text-emerald-400'
  if (ms < 1500) return 'text-cyan-400'
  if (ms < 3000) return 'text-amber-400'
  return 'text-rose-400'
}

function formatTime(timestamp: number): string {
  return new Date(timestamp).toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false
  })
}

function ActivityCard({ item, index }: { item: ActivityItem; index: number }) {
  const config = statusConfig[item.status]
  const Icon = config.icon

  return (
    <motion.div
      layout
      initial={{ opacity: 0, x: -20, height: 0 }}
      animate={{ opacity: 1, x: 0, height: 'auto' }}
      exit={{ opacity: 0, x: 20, height: 0 }}
      transition={{ 
        duration: 0.25,
        delay: index * 0.02,
        layout: { duration: 0.2 }
      }}
      className={cn(
        "relative flex items-center gap-3 p-3 rounded-xl",
        "border backdrop-blur-sm transition-all duration-200",
        "hover:bg-white/[0.02]",
        config.bg,
        config.border
      )}
    >
      <div className="relative shrink-0">
        <div className={cn(
          "w-8 h-8 rounded-lg flex items-center justify-center",
          config.bg
        )}>
          <Icon className={cn("w-4 h-4", config.color)} />
        </div>
        
        {item.status === 'success' && index === 0 && (
          <motion.div
            className={cn(
              "absolute inset-0 rounded-lg",
              config.bg
            )}
            animate={{ 
              scale: [1, 1.8, 1],
              opacity: [0.6, 0, 0.6]
            }}
            transition={{ duration: 2, repeat: Infinity }}
          />
        )}
      </div>

      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-white truncate">
            {item.model}
          </span>
          {item.endpoint && (
            <span className="text-xs text-zinc-600 truncate hidden sm:inline">
              {item.endpoint}
            </span>
          )}
        </div>
        <div className="flex items-center gap-3 mt-0.5 text-xs text-zinc-500">
          <span>{formatTime(item.timestamp)}</span>
          <span className="w-px h-3 bg-zinc-700" />
          <span>{item.auth_id.slice(0, 8)}...</span>
        </div>
      </div>

      <div className="flex items-center gap-4 shrink-0">
        <div className="text-right hidden sm:block">
          <p className="text-sm font-medium text-zinc-300">
            {item.tokens.toLocaleString()}
          </p>
          <p className="text-[10px] text-zinc-600 uppercase">tokens</p>
        </div>
        
        <div className="text-right min-w-[60px]">
          <p className={cn(
            "text-sm font-mono font-medium",
            getLatencyColor(item.latency_ms)
          )}>
            {item.latency_ms}ms
          </p>
          <p className="text-[10px] text-zinc-600 uppercase">latency</p>
        </div>
      </div>
    </motion.div>
  )
}

export function ActivityFeed({ items, maxItems = 10 }: ActivityFeedProps) {
  const displayItems = items.slice(0, maxItems)

  return (
    <div className="space-y-2 max-h-[450px] overflow-y-auto pr-1 scrollbar-thin">
      <AnimatePresence mode="popLayout" initial={false}>
        {displayItems.map((item, index) => (
          <ActivityCard key={item.id} item={item} index={index} />
        ))}
      </AnimatePresence>

      {displayItems.length === 0 && (
        <motion.div 
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="flex flex-col items-center justify-center py-12 text-zinc-500"
        >
          <Zap className="w-10 h-10 mb-3 text-zinc-700" />
          <p className="text-sm font-medium">No activity yet</p>
          <p className="text-xs text-zinc-600 mt-1">Requests will appear here in real-time</p>
        </motion.div>
      )}
    </div>
  )
}
