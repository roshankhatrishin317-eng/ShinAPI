'use client'

import { motion } from 'framer-motion'
import { cn } from '@/lib/utils'
import { AlertTriangle, XCircle, Clock } from 'lucide-react'

interface ErrorItem {
  id: string
  timestamp: number
  model: string
  error: string
  code: number
  auth_id: string
}

interface ErrorLogPanelProps {
  errors: ErrorItem[]
  maxItems?: number
}

const getErrorIcon = (code: number) => {
  if (code === 429) return Clock
  if (code >= 500) return XCircle
  return AlertTriangle
}

const getErrorColor = (code: number) => {
  if (code === 429) return 'text-amber-400'
  if (code >= 500) return 'text-red-400'
  return 'text-orange-400'
}

export function ErrorLogPanel({ errors, maxItems = 5 }: ErrorLogPanelProps) {
  const displayErrors = errors.slice(0, maxItems)

  return (
    <div className="space-y-2">
      {displayErrors.map((error, index) => {
        const Icon = getErrorIcon(error.code)
        const colorClass = getErrorColor(error.code)
        
        return (
          <motion.div
            key={error.id}
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: index * 0.05 }}
            className="p-3 rounded-lg bg-red-500/5 border border-red-500/20"
          >
            <div className="flex items-start gap-3">
              <Icon className={cn("w-4 h-4 mt-0.5 flex-shrink-0", colorClass)} />
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <span className={cn("text-xs font-mono px-1.5 py-0.5 rounded", colorClass, "bg-current/10")}>
                    {error.code}
                  </span>
                  <span className="text-xs text-zinc-500 truncate">{error.model}</span>
                  <span className="text-xs text-zinc-600 ml-auto">
                    {new Date(error.timestamp).toLocaleTimeString()}
                  </span>
                </div>
                <p className="text-xs text-zinc-400 line-clamp-2">{error.error}</p>
              </div>
            </div>
          </motion.div>
        )
      })}

      {displayErrors.length === 0 && (
        <div className="text-center py-6 text-zinc-500 text-sm">
          No errors
        </div>
      )}
    </div>
  )
}
