'use client'

import { motion } from 'framer-motion'
import { cn } from '@/lib/utils'
import { Wifi, WifiOff, RefreshCw, AlertCircle } from 'lucide-react'

type ConnectionState = 'connecting' | 'connected' | 'disconnected' | 'reconnecting'

interface ConnectionStatusProps {
  state: ConnectionState
  reconnectAttempts?: number
  lastMessageTime?: number | null
}

const stateConfig = {
  connecting: {
    icon: RefreshCw,
    label: 'Connecting',
    color: 'text-blue-400',
    bg: 'bg-blue-500/20',
    border: 'border-blue-500/30',
    animate: true
  },
  connected: {
    icon: Wifi,
    label: 'Live',
    color: 'text-emerald-400',
    bg: 'bg-emerald-500/20',
    border: 'border-emerald-500/30',
    animate: false
  },
  disconnected: {
    icon: WifiOff,
    label: 'Disconnected',
    color: 'text-red-400',
    bg: 'bg-red-500/20',
    border: 'border-red-500/30',
    animate: false
  },
  reconnecting: {
    icon: RefreshCw,
    label: 'Reconnecting',
    color: 'text-amber-400',
    bg: 'bg-amber-500/20',
    border: 'border-amber-500/30',
    animate: true
  }
}

export function ConnectionStatus({ state, reconnectAttempts = 0, lastMessageTime }: ConnectionStatusProps) {
  const config = stateConfig[state]
  const Icon = config.icon
  
  const timeSinceLastMessage = lastMessageTime 
    ? Math.floor((Date.now() - lastMessageTime) / 1000)
    : null

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.9 }}
      animate={{ opacity: 1, scale: 1 }}
      className={cn(
        "flex items-center gap-2 px-3 py-1.5 rounded-full border text-sm",
        config.bg,
        config.border
      )}
    >
      <motion.div
        animate={config.animate ? { rotate: 360 } : {}}
        transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
      >
        <Icon className={cn("w-4 h-4", config.color)} />
      </motion.div>
      
      <span className={cn("font-medium", config.color)}>
        {config.label}
        {state === 'reconnecting' && reconnectAttempts > 0 && ` (${reconnectAttempts})`}
      </span>

      {state === 'connected' && (
        <motion.div
          className="w-1.5 h-1.5 rounded-full bg-emerald-400"
          animate={{ opacity: [1, 0.3, 1] }}
          transition={{ duration: 1.5, repeat: Infinity }}
        />
      )}
    </motion.div>
  )
}
