'use client'

import { useState, useEffect, useCallback } from 'react'
import useSWR from 'swr'

interface MetricBucket {
  timestamp: string
  requests: number
  tokens: number
  input_tokens: number
  output_tokens: number
  avg_latency_ms: number
  success_count: number
  failure_count: number
  by_model?: Record<string, {
    requests: number
    tokens: number
    input_tokens: number
    output_tokens: number
    avg_latency_ms: number
  }>
}

interface HistoricalData {
  seconds?: MetricBucket[]
  minutes?: MetricBucket[]
  hours?: MetricBucket[]
  days?: MetricBucket[]
}

interface HistoricalSummary {
  total_requests: number
  total_tokens: number
  avg_tps: number
  avg_tpm: number
  peak_tps: number
  peak_tpm: number
  success_rate: number
  avg_latency_ms: number
}

interface MetricsResponse {
  success: boolean
  type?: string
  data: MetricBucket[]
  current_tps?: number
  current_tpm?: number
  current_tph?: number
  current_tpd?: number
}

interface SummaryResponse {
  success: boolean
  data: HistoricalSummary
}

// Database status checker
export function useDatabaseStatus() {
  const { data, error, isLoading, mutate } = useSWR(
    '/api/metrics/init',
    (url) => fetch(url).then(res => res.json()),
    { refreshInterval: 0, revalidateOnFocus: false }
  )

  const initialize = useCallback(async () => {
    const res = await fetch('/api/metrics/init', { method: 'POST' })
    const result = await res.json()
    mutate()
    return result
  }, [mutate])

  return {
    initialized: data?.initialized || false,
    tables: data?.tables || [],
    counts: data?.counts || {},
    missingTables: data?.missingTables || [],
    isLoading,
    error,
    initialize,
    refresh: mutate,
  }
}

// Fetcher that tries database first, then falls back to management API
const fetcher = async (url: string) => {
  // Try database API first
  try {
    const dbRes = await fetch(url)
    if (dbRes.ok) {
      const data = await dbRes.json()
      if (data.success && data.data && data.data.length > 0) {
        return data
      }
    }
  } catch (e) {
    console.warn('Database fetch failed, falling back to management API')
  }

  // Fallback to management API
  const key = typeof window !== 'undefined' ? localStorage.getItem('shinapi_mgmt_key') : null
  const mgmtUrl = url.replace('/api/metrics', '/v0/management/metrics')
  
  const res = await fetch(mgmtUrl, {
    headers: key ? { 'Authorization': `Bearer ${key}` } : {},
  })
  
  if (!res.ok) throw new Error('Failed to fetch')
  return res.json()
}

export function useHistoricalMetrics(range: string = '1h', refreshInterval: number = 5000) {
  const { data, error, isLoading, mutate } = useSWR<{ data: HistoricalSummary }>(
    `/api/metrics?type=summary&range=${range}`,
    fetcher,
    { refreshInterval }
  )

  return {
    data: null,
    summary: data?.data,
    range,
    isLoading,
    error,
    refresh: mutate,
  }
}

export function useTPSMetrics(granularity: string = 'second', refreshInterval: number = 3000) {
  const { data, error, isLoading, mutate } = useSWR<MetricsResponse>(
    `/api/metrics?type=tps&granularity=${granularity}`,
    fetcher,
    { refreshInterval }
  )

  return {
    currentTPS: data?.current_tps || 0,
    data: data?.data || [],
    isLoading,
    error,
    refresh: mutate,
  }
}

export function useTPMMetrics(granularity: string = 'minute', refreshInterval: number = 5000) {
  const { data, error, isLoading, mutate } = useSWR<MetricsResponse>(
    `/api/metrics?type=tpm&granularity=${granularity}`,
    fetcher,
    { refreshInterval }
  )

  return {
    currentTPM: data?.current_tpm || 0,
    data: data?.data || [],
    isLoading,
    error,
    refresh: mutate,
  }
}

export function useTPHMetrics(range: string = '24h', refreshInterval: number = 30000) {
  const { data, error, isLoading, mutate } = useSWR<MetricsResponse>(
    `/api/metrics?type=tph&range=${range}`,
    fetcher,
    { refreshInterval }
  )

  return {
    currentTPH: data?.current_tph || 0,
    data: data?.data || [],
    isLoading,
    error,
    refresh: mutate,
  }
}

export function useTPDMetrics(range: string = '30d', refreshInterval: number = 60000) {
  const { data, error, isLoading, mutate } = useSWR<MetricsResponse>(
    `/api/metrics?type=tpd&range=${range}`,
    fetcher,
    { refreshInterval }
  )

  return {
    currentTPD: data?.current_tpd || 0,
    data: data?.data || [],
    isLoading,
    error,
    refresh: mutate,
  }
}

export function useAllMetrics() {
  const tps = useTPSMetrics()
  const tpm = useTPMMetrics()
  const tph = useTPHMetrics()
  const tpd = useTPDMetrics()

  return {
    tps,
    tpm,
    tph,
    tpd,
    isLoading: tps.isLoading || tpm.isLoading || tph.isLoading || tpd.isLoading,
  }
}

// Hook to sync data from management API to database
// DISABLED: Go backend now handles database persistence directly
export function useMetricsSync() {
  const [syncing] = useState(false)
  const [lastSync] = useState<Date | null>(null)

  // Sync disabled - Go backend handles persistence
  const syncMetrics = useCallback(async () => {
    // No-op: Database sync is handled by Go backend
  }, [])

  return {
    syncing,
    lastSync,
    syncMetrics,
  }
}
