# Fix Plan

This plan focuses on backend quality, efficiency, and correctness (not security hardening).

## Scope
- Backend runtime correctness and performance in metrics, logging, and request tracking.
- CI workflow coverage gaps for tests/quality gates.
- Release and Docker pipeline correctness and efficiency.

## Backend fixes

### 1) WebSocket metrics hub reliability
- **Problem**: Potential deadlock + data race on client map.
- **Files**: `internal/api/websocket_hub.go`
- **Changes**:
  - Remove internal broadcast channel for metrics or make send non-blocking.
  - Use `Lock` for map mutation and `RLock` only for read-only iteration.
  - Ensure message broadcast does not block the main loop.
- **Why**: Prevent hub freezes under load; remove data races and map corruption.

### 2) Correct per-model latency aggregation
- **Problem**: Model latency averages are lost beyond per-second bucket.
- **Files**: `internal/usage/historical_metrics.go`
- **Changes**:
  - Track per-model latency sum + count in aggregations.
  - Compute weighted avg latency for per-model in minute/hour/day rollups.
- **Why**: Fix incorrect latency metrics in dashboards and DB.

### 3) Metrics DB flush control
- **Problem**: Unbounded goroutine creation on flush.
- **Files**: `internal/usage/metrics_db.go`
- **Changes**:
  - Introduce a single flush worker with a buffered signal channel.
  - Replace `go db.flush()` in `Record` with a non-blocking notify.
- **Why**: Prevent goroutine explosion and DB overload on high QPS.

### 4) Metrics DB Close idempotency
- **Problem**: `Close()` can panic on double-close.
- **Files**: `internal/usage/metrics_db.go`
- **Changes**:
  - Guard `close(db.done)` with `sync.Once` or a closed-channel pattern.
- **Why**: Robust shutdown with no panic during restarts or multiple closers.

### 5) Bound in-memory request details
- **Problem**: `RequestStatistics` grows unbounded, hurting memory and CPU.
- **Files**: `internal/usage/logger_plugin.go`, `internal/api/handlers/management/live_metrics.go`
- **Changes**:
  - Cap `modelStatsValue.Details` using ring buffer or max length.
  - Prefer `RealTimeTracker` for live metrics instead of full history scans.
- **Why**: Keep request metrics lightweight and predictable under load.

## CI workflow improvements

### 6) PR workflow quality gates
- **Problem**: PR workflow only builds one binary.
- **Files**: `.github/workflows/pr-test-build.yml`
- **Changes**:
  - Add `go test ./...`, `go vet ./...`.
  - Add `golangci-lint` or `staticcheck` job.
  - Add `gofmt` (or gofumpt) diff check.
  - Add `go mod tidy` check to detect module drift.
- **Why**: Catch regressions early and broaden compile/test coverage.

## Release + Docker pipeline improvements

### 7) Tag trigger alignment
- **Problem**: Release runs on all tags; Docker on `v*` only.
- **Files**: `.github/workflows/release.yaml`, `.github/workflows/docker-image.yml`
- **Changes**:
  - Align both workflows to the same tag pattern.
- **Why**: Ensure release artifacts and Docker images match.

### 8) Docker build caching + metadata
- **Problem**: No build cache and inconsistent version metadata.
- **Files**: `.github/workflows/docker-image.yml`
- **Changes**:
  - Add `cache-from`/`cache-to` for buildx.
  - Add labels for version/commit/build date.
  - Make Docker tag derived from the git tag, not `--dirty` describe.
- **Why**: Faster builds and consistent, traceable images.

## Suggested order of work
1) WebSocket hub reliability (deadlock + race).
2) Metrics correctness (latency aggregation + request detail bounds).
3) Metrics DB flush + Close idempotency.
4) CI quality gates in PR workflow.
5) Release/Docker alignment + caching.

## Verification steps
- Run `go test ./...` after backend changes.
- Run a local metrics stress test (simulated requests) to verify:
  - No hub deadlocks.
  - Latency averages show correct values for minute/hour/day.
  - Memory stable with bounded request details.
- Trigger CI and release workflow checks on a test tag.
