'use client'

import React from 'react'
import { motion } from 'framer-motion'
import { 
  Server, CheckCircle, XCircle, AlertTriangle, 
  Clock, Zap, Activity, Radio
} from 'lucide-react'
import { Badge } from '@/components/ui/badge'

interface ProviderStatus {
  name: string
  type: string
  healthy: boolean
  requests: number
  errors: number
  error_rate_percent: number
  avg_latency_ms: number
  p95_latency_ms: number
  credentials: number
  rate_limited: boolean
}

interface ProviderHealthProps {
  providers: ProviderStatus[]
  loading?: boolean
}

function formatNumber(num: number): string {
  if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
  return Math.round(num).toString()
}

function getStatusColor(provider: ProviderStatus): string {
  if (provider.rate_limited) return 'amber'
  if (!provider.healthy) return 'rose'
  if (provider.error_rate_percent > 10) return 'amber'
  return 'emerald'
}

function getStatusIcon(provider: ProviderStatus) {
  if (provider.rate_limited) return <AlertTriangle className="w-4 h-4 text-amber-400" />
  if (!provider.healthy) return <XCircle className="w-4 h-4 text-rose-400" />
  return <CheckCircle className="w-4 h-4 text-emerald-400" />
}

export function ProviderHealthPanel({ providers, loading }: ProviderHealthProps) {
  if (loading) {
    return (
      <div className="flex items-center justify-center h-40">
        <motion.div
          className="w-8 h-8 border-2 border-zinc-600 border-t-blue-500 rounded-full"
          animate={{ rotate: 360 }}
          transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
        />
      </div>
    )
  }

  if (!providers || providers.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-40 text-zinc-500">
        <Server className="w-8 h-8 mb-2 opacity-50" />
        <p className="text-sm">No providers configured</p>
      </div>
    )
  }

  const healthyCount = providers.filter(p => p.healthy && !p.rate_limited).length
  const unhealthyCount = providers.length - healthyCount

  return (
    <div className="space-y-4">
      {/* Summary */}
      <div className="flex items-center justify-between pb-3 border-b border-zinc-800">
        <div className="flex items-center gap-2">
          <Radio className="w-4 h-4 text-blue-400" />
          <span className="text-xs text-zinc-400">
            {healthyCount} healthy, {unhealthyCount} issues
          </span>
        </div>
        <div className="flex gap-1">
          {healthyCount > 0 && (
            <Badge variant="success" className="text-[10px]">
              {healthyCount} OK
            </Badge>
          )}
          {unhealthyCount > 0 && (
            <Badge variant="error" className="text-[10px]">
              {unhealthyCount} DOWN
            </Badge>
          )}
        </div>
      </div>

      {/* Provider list */}
      <div className="space-y-2 max-h-[280px] overflow-y-auto pr-1 scrollbar-thin scrollbar-thumb-zinc-700 scrollbar-track-transparent">
        {providers.map((provider, index) => (
          <motion.div
            key={provider.name}
            initial={{ opacity: 0, x: -10 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: index * 0.05 }}
            className={`
              p-3 rounded-lg border transition-all
              ${provider.healthy && !provider.rate_limited
                ? 'border-emerald-500/20 bg-emerald-500/5'
                : provider.rate_limited
                ? 'border-amber-500/20 bg-amber-500/5'
                : 'border-rose-500/20 bg-rose-500/5'
              }
            `}
          >
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-2">
                {getStatusIcon(provider)}
                <span className="text-sm font-medium text-zinc-200">
                  {provider.name}
                </span>
                <Badge variant="info" className="text-[10px]">
                  {provider.type}
                </Badge>
              </div>
              {provider.rate_limited && (
                <Badge variant="warning" className="text-[10px]">
                  RATE LIMITED
                </Badge>
              )}
            </div>

            <div className="grid grid-cols-4 gap-2 text-xs">
              <div>
                <div className="text-zinc-500">Requests</div>
                <div className="text-zinc-200 font-medium">
                  {formatNumber(provider.requests)}
                </div>
              </div>
              <div>
                <div className="text-zinc-500">Errors</div>
                <div className={`font-medium ${
                  provider.errors > 0 ? 'text-rose-400' : 'text-zinc-200'
                }`}>
                  {formatNumber(provider.errors)}
                </div>
              </div>
              <div>
                <div className="text-zinc-500">Error Rate</div>
                <div className={`font-medium ${
                  provider.error_rate_percent > 10 
                    ? 'text-rose-400' 
                    : provider.error_rate_percent > 5 
                    ? 'text-amber-400' 
                    : 'text-emerald-400'
                }`}>
                  {provider.error_rate_percent.toFixed(1)}%
                </div>
              </div>
              <div>
                <div className="text-zinc-500">Latency</div>
                <div className="text-zinc-200 font-medium">
                  {provider.avg_latency_ms.toFixed(0)}ms
                </div>
              </div>
            </div>

            {provider.credentials > 0 && (
              <div className="mt-2 flex items-center gap-1 text-[10px] text-zinc-500">
                <Activity className="w-3 h-3" />
                <span>{provider.credentials} credential{provider.credentials > 1 ? 's' : ''}</span>
              </div>
            )}
          </motion.div>
        ))}
      </div>
    </div>
  )
}

export function ProviderHealthSummary({ providers }: { providers: ProviderStatus[] }) {
  if (!providers || providers.length === 0) return null

  const healthy = providers.filter(p => p.healthy && !p.rate_limited)
  const unhealthy = providers.filter(p => !p.healthy || p.rate_limited)
  const totalRequests = providers.reduce((sum, p) => sum + p.requests, 0)
  const avgErrorRate = providers.reduce((sum, p) => sum + p.error_rate_percent, 0) / providers.length

  return (
    <div className="grid grid-cols-4 gap-4">
      <div className="text-center">
        <div className="text-2xl font-bold text-zinc-100">{providers.length}</div>
        <div className="text-xs text-zinc-500">Providers</div>
      </div>
      <div className="text-center">
        <div className="text-2xl font-bold text-emerald-400">{healthy.length}</div>
        <div className="text-xs text-zinc-500">Healthy</div>
      </div>
      <div className="text-center">
        <div className={`text-2xl font-bold ${unhealthy.length > 0 ? 'text-rose-400' : 'text-zinc-400'}`}>
          {unhealthy.length}
        </div>
        <div className="text-xs text-zinc-500">Issues</div>
      </div>
      <div className="text-center">
        <div className={`text-2xl font-bold ${
          avgErrorRate > 10 ? 'text-rose-400' : avgErrorRate > 5 ? 'text-amber-400' : 'text-emerald-400'
        }`}>
          {avgErrorRate.toFixed(1)}%
        </div>
        <div className="text-xs text-zinc-500">Avg Errors</div>
      </div>
    </div>
  )
}
