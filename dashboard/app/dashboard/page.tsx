// Server-Side Rendered Dashboard
// No 'use client' - renders entirely on server, sends HTML to browser
// Dramatically reduces browser memory usage (550MB → ~100-150MB)

import { Suspense } from 'react'
import { cookies, headers } from 'next/headers'
import { BarChart3, Settings, LogOut, Command } from 'lucide-react'
import { cn } from '@/lib/utils'

// Server components
import { MetricsCards, StatsCards, LatencyCards, UptimeCard } from '@/components/server/MetricsCards'
import { ChartsSection, ThroughputGauges } from '@/components/server/ChartsSection'

// Minimal client component for live updates
import { LiveIndicator, LiveMetricsBar } from '@/components/client/LiveIndicator'

// Import CSS animations
import '@/styles/animations.css'

interface Metrics {
  rpm: number
  tpm: number
  tps: number
  total_requests: number
  total_tokens: number
  total_success: number
  total_failed: number
  success_rate: number
  avg_latency_ms: number
  p50_latency_ms: number
  p95_latency_ms: number
  p99_latency_ms: number
  uptime_seconds: number
  model_stats: Record<string, { requests: number; tokens: number }>
  recent_requests?: any[]
  recent_errors?: any[]
}

// Server-side data fetching
async function getMetrics(authKey?: string): Promise<Metrics | null> {
  try {
    // Use environment variable for server-side auth
    const key = authKey || process.env.SHINAPI_MGMT_KEY
    if (!key) return null

    const baseUrl = process.env.SHINAPI_URL || 'http://localhost:8317'
    
    const res = await fetch(`${baseUrl}/v0/management/live-metrics`, {
      headers: { 
        'Authorization': `Bearer ${key}`,
        'Content-Type': 'application/json'
      },
      // Revalidate every 2 seconds for near-real-time data
      next: { revalidate: 2 }
    })

    if (!res.ok) return null
    return res.json()
  } catch (error) {
    console.error('Failed to fetch metrics:', error)
    return null
  }
}

// Header component (server-rendered)
function DashboardHeader() {
  return (
    <header className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-8 animate-fade-in-down">
      <div className="flex items-center gap-3">
        <div className="p-2.5 rounded-xl bg-gradient-to-br from-blue-500/20 to-violet-500/20 border border-white/10">
          <BarChart3 className="w-6 h-6 text-blue-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">ShinAPI Dashboard</h1>
          <p className="text-sm text-zinc-500">Real-time API metrics & analytics</p>
        </div>
      </div>

      <div className="flex items-center gap-2">
        {/* Live indicator (client component) */}
        <Suspense fallback={<div className="w-20 h-4 bg-zinc-800 rounded animate-skeleton" />}>
          <LiveIndicator />
        </Suspense>
      </div>
    </header>
  )
}

// Section wrapper with suspense boundary
function Section({ 
  children, 
  fallback 
}: { 
  children: React.ReactNode
  fallback?: React.ReactNode
}) {
  return (
    <section className="mb-6">
      <Suspense fallback={fallback || <SectionSkeleton />}>
        {children}
      </Suspense>
    </section>
  )
}

function SectionSkeleton() {
  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
      {Array.from({ length: 4 }).map((_, i) => (
        <div 
          key={i}
          className="h-24 rounded-xl border border-zinc-800/50 bg-zinc-900/50 animate-skeleton"
        />
      ))}
    </div>
  )
}

// Main dashboard page (Server Component)
export default async function DashboardPage() {
  const metrics = await getMetrics()

  // If no metrics (no auth key), show minimal page
  // Client-side auth will be handled by LiveIndicator
  if (!metrics) {
    return (
      <div className="min-h-screen bg-zinc-950 text-white">
        <Background />
        <div className="max-w-7xl mx-auto p-4 lg:p-8">
          <DashboardHeader />
          <div className="flex flex-col items-center justify-center min-h-[60vh]">
            <div className="text-center animate-fade-in-up">
              <BarChart3 className="w-16 h-16 text-zinc-700 mx-auto mb-4" />
              <h2 className="text-xl font-semibold text-zinc-400 mb-2">Connecting to API...</h2>
              <p className="text-sm text-zinc-600 mb-6">
                Set SHINAPI_MGMT_KEY environment variable or use client-side auth
              </p>
              <div className="flex items-center gap-2 justify-center">
                <span className="w-2 h-2 rounded-full bg-amber-400 animate-pulse-dot" />
                <span className="text-xs text-zinc-500">Waiting for connection</span>
              </div>
            </div>
          </div>
        </div>
        {/* Client-side live bar for real-time updates */}
        <Suspense fallback={null}>
          <LiveMetricsBar />
        </Suspense>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-zinc-950 text-white pb-16">
      <Background />
      
      <div className="max-w-7xl mx-auto p-4 lg:p-8">
        <DashboardHeader />

        {/* Real-time Metrics */}
        <Section>
          <MetricsCards metrics={metrics} />
        </Section>

        {/* Stats Overview */}
        <Section>
          <StatsCards metrics={metrics} />
        </Section>

        {/* Throughput Gauges */}
        <Section>
          <ThroughputGauges 
            tps={metrics.tps}
            tpm={metrics.tpm}
            tph={0}
            tpd={0}
          />
        </Section>

        {/* Charts */}
        <Section>
          <ChartsSection 
            modelStats={metrics.model_stats}
            rpmHistory={[]}
            tpmHistory={[]}
          />
        </Section>

        {/* Latency */}
        <Section>
          <LatencyCards metrics={metrics} />
        </Section>

        {/* Uptime */}
        <Section>
          <div className="max-w-sm">
            <UptimeCard metrics={metrics} />
          </div>
        </Section>
      </div>

      {/* Live metrics bar at bottom (client component) */}
      <Suspense fallback={null}>
        <LiveMetricsBar />
      </Suspense>
    </div>
  )
}

// Background gradients (pure CSS, no JS)
function Background() {
  return (
    <div className="fixed inset-0 -z-10 overflow-hidden pointer-events-none">
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top_left,rgba(59,130,246,0.12),transparent_50%)]" />
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_bottom_right,rgba(139,92,246,0.12),transparent_50%)]" />
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_center,rgba(6,182,212,0.05),transparent_70%)]" />
      
      {/* Grid pattern */}
      <svg className="absolute inset-0 w-full h-full opacity-[0.015]">
        <defs>
          <pattern id="grid" width="60" height="60" patternUnits="userSpaceOnUse">
            <path d="M 60 0 L 0 0 0 60" fill="none" stroke="white" strokeWidth="0.5"/>
          </pattern>
        </defs>
        <rect width="100%" height="100%" fill="url(#grid)" />
      </svg>
    </div>
  )
}

// Metadata for SEO
export const metadata = {
  title: 'ShinAPI Dashboard - Real-time Metrics',
  description: 'Monitor your AI API gateway with real-time metrics and analytics',
}
