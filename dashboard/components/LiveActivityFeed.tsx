'use client'

import { motion, AnimatePresence } from 'framer-motion'
import { cn } from '@/lib/utils'

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

interface LiveActivityFeedProps {
  items: ActivityItem[]
  maxItems?: number
}

const statusColors = {
  success: 'bg-emerald-500',
  error: 'bg-red-500',
  rate_limited: 'bg-amber-500'
}

const statusBg = {
  success: 'bg-emerald-500/10 border-emerald-500/20',
  error: 'bg-red-500/10 border-red-500/20',
  rate_limited: 'bg-amber-500/10 border-amber-500/20'
}

export function LiveActivityFeed({ items, maxItems = 10 }: LiveActivityFeedProps) {
  const displayItems = items.slice(0, maxItems)

  return (
    <div className="space-y-2 max-h-[400px] overflow-y-auto pr-2 scrollbar-thin">
      <AnimatePresence mode="popLayout">
        {displayItems.map((item, index) => (
          <motion.div
            key={item.id}
            initial={{ opacity: 0, x: -20, height: 0 }}
            animate={{ opacity: 1, x: 0, height: 'auto' }}
            exit={{ opacity: 0, x: 20, height: 0 }}
            transition={{ duration: 0.2, delay: index * 0.02 }}
            className={cn(
              "flex items-center gap-3 p-3 rounded-lg border backdrop-blur-sm",
              statusBg[item.status]
            )}
          >
            {/* Status indicator */}
            <div className="relative">
              <div className={cn("w-2 h-2 rounded-full", statusColors[item.status])} />
              {item.status === 'success' && index === 0 && (
                <motion.div
                  className={cn("absolute inset-0 rounded-full", statusColors[item.status])}
                  animate={{ scale: [1, 2, 1], opacity: [0.5, 0, 0.5] }}
                  transition={{ duration: 1.5, repeat: Infinity }}
                />
              )}
            </div>

            {/* Model */}
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-white truncate">{item.model}</p>
              <p className="text-xs text-zinc-500">
                {new Date(item.timestamp).toLocaleTimeString()}
              </p>
            </div>

            {/* Metrics */}
            <div className="flex items-center gap-4 text-xs">
              <div className="text-right">
                <p className="text-zinc-400">{item.tokens.toLocaleString()}</p>
                <p className="text-zinc-600">tokens</p>
              </div>
              <div className="text-right min-w-[50px]">
                <p className={cn(
                  "font-mono",
                  item.latency_ms < 1000 ? "text-emerald-400" :
                  item.latency_ms < 3000 ? "text-amber-400" : "text-red-400"
                )}>
                  {item.latency_ms}ms
                </p>
                <p className="text-zinc-600">latency</p>
              </div>
            </div>
          </motion.div>
        ))}
      </AnimatePresence>

      {displayItems.length === 0 && (
        <div className="text-center py-8 text-zinc-500">
          No activity yet
        </div>
      )}
    </div>
  )
}
