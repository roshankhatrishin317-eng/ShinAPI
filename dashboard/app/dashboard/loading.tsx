// Loading skeleton for SSR streaming
// Shows immediately while server fetches data

import { cn } from '@/lib/utils'

function SkeletonCard({ className }: { className?: string }) {
  return (
    <div className={cn(
      'rounded-xl border border-zinc-800/50 bg-zinc-900/50 p-4',
      'animate-skeleton',
      className
    )}>
      <div className="flex items-center gap-2 mb-3">
        <div className="w-4 h-4 rounded bg-zinc-800" />
        <div className="w-20 h-3 rounded bg-zinc-800" />
      </div>
      <div className="w-24 h-8 rounded bg-zinc-800" />
    </div>
  )
}

function SkeletonChart({ className }: { className?: string }) {
  return (
    <div className={cn(
      'rounded-xl border border-zinc-800/50 bg-zinc-900/50 p-4',
      'animate-skeleton',
      className
    )}>
      <div className="flex items-center gap-2 mb-4">
        <div className="w-5 h-5 rounded bg-zinc-800" />
        <div className="w-28 h-4 rounded bg-zinc-800" />
      </div>
      <div className="h-32 flex items-end justify-between gap-1">
        {Array.from({ length: 12 }).map((_, i) => (
          <div 
            key={i}
            className="flex-1 rounded-t bg-zinc-800"
            style={{ height: `${20 + Math.random() * 60}%` }}
          />
        ))}
      </div>
    </div>
  )
}

export default function DashboardLoading() {
  return (
    <div className="min-h-screen bg-zinc-950 text-white">
      {/* Background */}
      <div className="fixed inset-0 -z-10 overflow-hidden">
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top_left,rgba(59,130,246,0.08),transparent_50%)]" />
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_bottom_right,rgba(139,92,246,0.08),transparent_50%)]" />
      </div>

      <div className="max-w-7xl mx-auto p-4 lg:p-8">
        {/* Header Skeleton */}
        <header className="flex items-center justify-between mb-8">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-zinc-800 animate-skeleton" />
            <div>
              <div className="w-32 h-6 rounded bg-zinc-800 animate-skeleton mb-1" />
              <div className="w-48 h-4 rounded bg-zinc-800 animate-skeleton" />
            </div>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-24 h-8 rounded-lg bg-zinc-800 animate-skeleton" />
            <div className="w-24 h-8 rounded-lg bg-zinc-800 animate-skeleton" />
          </div>
        </header>

        {/* Metrics Cards Skeleton */}
        <section className="mb-6">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <SkeletonCard />
            <SkeletonCard />
            <SkeletonCard />
            <SkeletonCard />
          </div>
        </section>

        {/* Stats Cards Skeleton */}
        <section className="mb-6">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <SkeletonCard />
            <SkeletonCard />
            <SkeletonCard />
            <SkeletonCard />
          </div>
        </section>

        {/* Throughput Skeleton */}
        <section className="mb-6">
          <div className="rounded-xl border border-zinc-800/50 bg-zinc-900/50 p-6 animate-skeleton">
            <div className="flex items-center gap-3 mb-6">
              <div className="w-10 h-10 rounded-lg bg-zinc-800" />
              <div>
                <div className="w-40 h-5 rounded bg-zinc-800 mb-1" />
                <div className="w-56 h-3 rounded bg-zinc-800" />
              </div>
            </div>
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
              {Array.from({ length: 4 }).map((_, i) => (
                <div key={i} className="flex flex-col items-center">
                  <div className="w-24 h-14 rounded bg-zinc-800" />
                  <div className="w-12 h-3 rounded bg-zinc-800 mt-2" />
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* Charts Skeleton */}
        <section className="mb-6">
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
            <SkeletonChart />
            <SkeletonChart />
            <SkeletonChart />
          </div>
        </section>

        {/* Latency Skeleton */}
        <section>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <SkeletonCard />
            <SkeletonCard />
            <SkeletonCard />
            <SkeletonCard />
          </div>
        </section>
      </div>
    </div>
  )
}

// Export individual skeleton components for Suspense boundaries
export { SkeletonCard, SkeletonChart }
