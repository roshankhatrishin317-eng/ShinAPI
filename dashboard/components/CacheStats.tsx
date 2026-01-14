'use client'

import React from 'react'
import { motion } from 'framer-motion'
import { Database, HardDrive, Layers, Zap } from 'lucide-react'
import { RingProgress } from '@mantine/core'

interface CacheStats {
  lru: {
    hits: number
    misses: number
    size: number
    capacity: number
    hit_rate_percent: number
  }
  semantic?: {
    enabled: boolean
    hits: number
    misses: number
    index_size: number
    hit_rate_percent: number
  }
  streaming?: {
    enabled: boolean
    entries: number
    total_events: number
    total_size_bytes: number
    hit_rate_percent: number
  }
  redis?: {
    enabled: boolean
    connected: boolean
    hits: number
    misses: number
    errors: number
    hit_rate_percent: number
    last_latency_ms: number
  }
}

interface CacheStatsProps {
  stats: CacheStats | null
  loading?: boolean
}

function formatBytes(bytes: number): string {
  if (bytes >= 1024 * 1024 * 1024) return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
  if (bytes >= 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  if (bytes >= 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return bytes + ' B'
}

function formatNumber(num: number): string {
  if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
  return Math.round(num).toString()
}

function getHitRateColor(rate: number): string {
  if (rate >= 80) return '#10b981' // emerald
  if (rate >= 50) return '#f59e0b' // amber
  return '#ef4444' // rose
}

export function CacheStatsPanel({ stats, loading }: CacheStatsProps) {
  if (loading || !stats) {
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

  const caches = [
    {
      name: 'LRU Cache',
      icon: Database,
      color: 'blue',
      enabled: true,
      hitRate: stats.lru.hit_rate_percent,
      hits: stats.lru.hits,
      misses: stats.lru.misses,
      extra: `${stats.lru.size}/${stats.lru.capacity} entries`,
    },
    ...(stats.semantic?.enabled ? [{
      name: 'Semantic',
      icon: Layers,
      color: 'purple',
      enabled: true,
      hitRate: stats.semantic.hit_rate_percent,
      hits: stats.semantic.hits,
      misses: stats.semantic.misses,
      extra: `${stats.semantic.index_size} indexed`,
    }] : []),
    ...(stats.streaming?.enabled ? [{
      name: 'Streaming',
      icon: Zap,
      color: 'cyan',
      enabled: true,
      hitRate: stats.streaming.hit_rate_percent,
      hits: 0,
      misses: 0,
      extra: `${stats.streaming.entries} cached (${formatBytes(stats.streaming.total_size_bytes)})`,
    }] : []),
    ...(stats.redis?.enabled ? [{
      name: 'Redis',
      icon: HardDrive,
      color: 'rose',
      enabled: stats.redis.connected,
      hitRate: stats.redis.hit_rate_percent,
      hits: stats.redis.hits,
      misses: stats.redis.misses,
      extra: stats.redis.connected 
        ? `${stats.redis.last_latency_ms.toFixed(1)}ms latency` 
        : 'Disconnected',
    }] : []),
  ]

  return (
    <div className="space-y-3">
      {caches.map((cache, index) => (
        <motion.div
          key={cache.name}
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: index * 0.1 }}
          className="flex items-center gap-4 p-3 rounded-lg bg-zinc-900/50 border border-zinc-800"
        >
          <RingProgress
            size={50}
            thickness={4}
            roundCaps
            sections={[{ value: cache.hitRate, color: getHitRateColor(cache.hitRate) }]}
            label={
              <div className="text-center">
                <span className="text-[10px] font-bold text-zinc-200">
                  {cache.hitRate.toFixed(0)}%
                </span>
              </div>
            }
          />
          
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <cache.icon className={`w-4 h-4 text-${cache.color}-400`} />
              <span className="text-sm font-medium text-zinc-200">{cache.name}</span>
              {!cache.enabled && (
                <span className="text-[10px] px-1.5 py-0.5 rounded bg-rose-500/20 text-rose-400">
                  OFF
                </span>
              )}
            </div>
            <div className="text-xs text-zinc-500 mt-0.5">{cache.extra}</div>
          </div>

          <div className="text-right text-xs">
            <div className="text-emerald-400">{formatNumber(cache.hits)} hits</div>
            <div className="text-zinc-500">{formatNumber(cache.misses)} miss</div>
          </div>
        </motion.div>
      ))}

      {caches.length === 1 && (
        <div className="text-center text-xs text-zinc-600 py-2">
          Enable semantic, streaming, or Redis cache for more options
        </div>
      )}
    </div>
  )
}

export function CacheHitRateRing({ hitRate }: { hitRate: number }) {
  return (
    <RingProgress
      size={80}
      thickness={8}
      roundCaps
      sections={[{ value: hitRate, color: getHitRateColor(hitRate) }]}
      label={
        <div className="text-center">
          <span className="text-lg font-bold text-zinc-200">
            {hitRate.toFixed(0)}%
          </span>
          <div className="text-[10px] text-zinc-500">Hit Rate</div>
        </div>
      }
    />
  )
}
