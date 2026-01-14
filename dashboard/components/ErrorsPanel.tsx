'use client'

import { motion } from 'framer-motion'
import { cn } from '@/lib/utils'
import { AlertTriangle, XCircle, Clock, ChevronRight, AlertCircle } from 'lucide-react'

interface ErrorItem {
  id: string
  timestamp: number
  model: string
  error: string
  code: number
  auth_id: string
}

interface ErrorsPanelProps {
  errors: ErrorItem[]
  maxItems?: number
}

function getErrorConfig(code: number) {
  if (code === 429) {
    return {
      icon: Clock,
      color: 'text-amber-400',
      bg: 'bg-amber-500/10',
      border: 'border-amber-500/20',
      label: 'Rate Limited'
    }
  }
  if (code >= 500) {
    return {
      icon: XCircle,
      color: 'text-rose-400',
      bg: 'bg-rose-500/10',
      border: 'border-rose-500/20',
      label: 'Server Error'
    }
  }
  return {
    icon: AlertTriangle,
    color: 'text-orange-400',
    bg: 'bg-orange-500/10',
    border: 'border-orange-500/20',
    label: 'Client Error'
  }
}

function ErrorCard({ error, index }: { error: ErrorItem; index: number }) {
  const config = getErrorConfig(error.code)
  const Icon = config.icon

  return (
    <motion.div
      initial={{ opacity: 0, y: -10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: index * 0.05 }}
      className={cn(
        "p-3 rounded-xl border transition-all duration-200",
        "hover:bg-white/[0.02] cursor-pointer",
        config.bg,
        config.border
      )}
    >
      <div className="flex items-start gap-3">
        <div className={cn(
          "shrink-0 w-8 h-8 rounded-lg flex items-center justify-center",
          config.bg
        )}>
          <Icon className={cn("w-4 h-4", config.color)} />
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className={cn(
              "px-2 py-0.5 rounded-md text-xs font-mono font-medium",
              config.bg, config.color
            )}>
              {error.code}
            </span>
            <span className="text-xs text-zinc-500 truncate">{error.model}</span>
            <span className="text-xs text-zinc-600 ml-auto shrink-0">
              {new Date(error.timestamp).toLocaleTimeString('en-US', {
                hour: '2-digit',
                minute: '2-digit',
                second: '2-digit',
                hour12: false
              })}
            </span>
          </div>
          
          <p className="text-sm text-zinc-400 line-clamp-2">{error.error}</p>
        </div>
      </div>
    </motion.div>
  )
}

export function ErrorsPanel({ errors, maxItems = 5 }: ErrorsPanelProps) {
  const displayErrors = errors.slice(0, maxItems)

  return (
    <div className="space-y-2">
      {displayErrors.map((error, index) => (
        <ErrorCard key={error.id} error={error} index={index} />
      ))}

      {displayErrors.length === 0 && (
        <motion.div 
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="flex flex-col items-center justify-center py-10 text-zinc-500"
        >
          <div className="w-12 h-12 rounded-full bg-emerald-500/10 flex items-center justify-center mb-3">
            <AlertCircle className="w-6 h-6 text-emerald-500" />
          </div>
          <p className="text-sm font-medium text-emerald-400">All systems operational</p>
          <p className="text-xs text-zinc-600 mt-1">No errors in the current session</p>
        </motion.div>
      )}
    </div>
  )
}
