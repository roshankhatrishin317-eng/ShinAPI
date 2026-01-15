"use server";

import { neon } from "@neondatabase/serverless";

const sql = neon(process.env.DATABASE_URL!);

// Initialize database tables
export async function initDatabase() {
  try {
    // Metrics snapshots table - stores periodic snapshots of metrics
    await sql`
      CREATE TABLE IF NOT EXISTS metrics_snapshots (
        id SERIAL PRIMARY KEY,
        timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        granularity VARCHAR(20) NOT NULL, -- 'second', 'minute', 'hour', 'day'
        requests BIGINT NOT NULL DEFAULT 0,
        tokens BIGINT NOT NULL DEFAULT 0,
        input_tokens BIGINT NOT NULL DEFAULT 0,
        output_tokens BIGINT NOT NULL DEFAULT 0,
        success_count BIGINT NOT NULL DEFAULT 0,
        failure_count BIGINT NOT NULL DEFAULT 0,
        avg_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
      )
    `;

    // Create index for faster queries
    await sql`
      CREATE INDEX IF NOT EXISTS idx_metrics_snapshots_timestamp 
      ON metrics_snapshots(timestamp DESC)
    `;

    await sql`
      CREATE INDEX IF NOT EXISTS idx_metrics_snapshots_granularity 
      ON metrics_snapshots(granularity, timestamp DESC)
    `;

    // Model metrics table - stores per-model breakdown
    await sql`
      CREATE TABLE IF NOT EXISTS model_metrics (
        id SERIAL PRIMARY KEY,
        snapshot_id INTEGER REFERENCES metrics_snapshots(id) ON DELETE CASCADE,
        model_name VARCHAR(255) NOT NULL,
        requests BIGINT NOT NULL DEFAULT 0,
        tokens BIGINT NOT NULL DEFAULT 0,
        input_tokens BIGINT NOT NULL DEFAULT 0,
        output_tokens BIGINT NOT NULL DEFAULT 0,
        avg_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0
      )
    `;

    await sql`
      CREATE INDEX IF NOT EXISTS idx_model_metrics_snapshot 
      ON model_metrics(snapshot_id)
    `;

    await sql`
      CREATE INDEX IF NOT EXISTS idx_model_metrics_model 
      ON model_metrics(model_name)
    `;

    // Daily aggregates table - pre-computed daily stats for faster queries
    await sql`
      CREATE TABLE IF NOT EXISTS daily_aggregates (
        id SERIAL PRIMARY KEY,
        date DATE NOT NULL UNIQUE,
        total_requests BIGINT NOT NULL DEFAULT 0,
        total_tokens BIGINT NOT NULL DEFAULT 0,
        total_input_tokens BIGINT NOT NULL DEFAULT 0,
        total_output_tokens BIGINT NOT NULL DEFAULT 0,
        success_count BIGINT NOT NULL DEFAULT 0,
        failure_count BIGINT NOT NULL DEFAULT 0,
        avg_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
        peak_tps DOUBLE PRECISION NOT NULL DEFAULT 0,
        peak_tpm BIGINT NOT NULL DEFAULT 0,
        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
      )
    `;

    await sql`
      CREATE INDEX IF NOT EXISTS idx_daily_aggregates_date 
      ON daily_aggregates(date DESC)
    `;

    // Hourly aggregates table
    await sql`
      CREATE TABLE IF NOT EXISTS hourly_aggregates (
        id SERIAL PRIMARY KEY,
        hour_start TIMESTAMPTZ NOT NULL UNIQUE,
        total_requests BIGINT NOT NULL DEFAULT 0,
        total_tokens BIGINT NOT NULL DEFAULT 0,
        total_input_tokens BIGINT NOT NULL DEFAULT 0,
        total_output_tokens BIGINT NOT NULL DEFAULT 0,
        success_count BIGINT NOT NULL DEFAULT 0,
        failure_count BIGINT NOT NULL DEFAULT 0,
        avg_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
      )
    `;

    await sql`
      CREATE INDEX IF NOT EXISTS idx_hourly_aggregates_hour 
      ON hourly_aggregates(hour_start DESC)
    `;

    return { success: true, message: "Database initialized successfully" };
  } catch (error) {
    console.error("Database initialization error:", error);
    return { success: false, message: String(error) };
  }
}

// Insert a metrics snapshot
export async function insertMetricsSnapshot(data: {
  granularity: "second" | "minute" | "hour" | "day";
  timestamp?: Date;
  requests: number;
  tokens: number;
  inputTokens: number;
  outputTokens: number;
  successCount: number;
  failureCount: number;
  avgLatencyMs: number;
  modelMetrics?: Array<{
    modelName: string;
    requests: number;
    tokens: number;
    inputTokens: number;
    outputTokens: number;
    avgLatencyMs: number;
  }>;
}) {
  try {
    const timestamp = data.timestamp || new Date();

    const result = await sql`
      INSERT INTO metrics_snapshots (
        timestamp, granularity, requests, tokens, input_tokens, output_tokens,
        success_count, failure_count, avg_latency_ms
      ) VALUES (
        ${timestamp.toISOString()}, ${data.granularity}, ${data.requests}, ${data.tokens},
        ${data.inputTokens}, ${data.outputTokens}, ${data.successCount}, 
        ${data.failureCount}, ${data.avgLatencyMs}
      )
      RETURNING id
    `;

    const snapshotId = result[0]?.id;

    // Insert model metrics if provided
    if (snapshotId && data.modelMetrics && data.modelMetrics.length > 0) {
      for (const model of data.modelMetrics) {
        await sql`
          INSERT INTO model_metrics (
            snapshot_id, model_name, requests, tokens, input_tokens, 
            output_tokens, avg_latency_ms
          ) VALUES (
            ${snapshotId}, ${model.modelName}, ${model.requests}, ${model.tokens},
            ${model.inputTokens}, ${model.outputTokens}, ${model.avgLatencyMs}
          )
        `;
      }
    }

    return { success: true, id: snapshotId };
  } catch (error) {
    console.error("Insert metrics error:", error);
    return { success: false, error: String(error) };
  }
}

// Get metrics by time range
export async function getMetricsByRange(
  granularity: "second" | "minute" | "hour" | "day",
  limit: number = 60
) {
  try {
    const data = await sql`
      SELECT 
        id, timestamp, granularity, requests, tokens, input_tokens, output_tokens,
        success_count, failure_count, avg_latency_ms
      FROM metrics_snapshots
      WHERE granularity = ${granularity}
      ORDER BY timestamp DESC
      LIMIT ${limit}
    `;

    return { success: true, data: data.reverse() };
  } catch (error) {
    console.error("Get metrics error:", error);
    return { success: false, data: [], error: String(error) };
  }
}

// Get TPS data (last N seconds)
export async function getTPSData(seconds: number = 60) {
  try {
    const data = await sql`
      SELECT 
        timestamp, requests, tokens, avg_latency_ms,
        success_count, failure_count
      FROM metrics_snapshots
      WHERE granularity = 'second'
        AND timestamp >= NOW() - INTERVAL '${seconds} seconds'
      ORDER BY timestamp ASC
    `;

    // Calculate current TPS (average of last 10 seconds)
    const recent = data.slice(-10);
    const currentTPS = recent.length > 0
      ? recent.reduce((sum, r) => sum + Number(r.requests), 0) / recent.length
      : 0;

    return { 
      success: true, 
      data, 
      currentTPS,
      total: data.reduce((sum, r) => sum + Number(r.requests), 0)
    };
  } catch (error) {
    console.error("Get TPS error:", error);
    return { success: false, data: [], currentTPS: 0, error: String(error) };
  }
}

// Get TPM data (last N minutes)
export async function getTPMData(minutes: number = 60) {
  try {
    const data = await sql`
      SELECT 
        timestamp, requests, tokens, input_tokens, output_tokens,
        avg_latency_ms, success_count, failure_count
      FROM metrics_snapshots
      WHERE granularity = 'minute'
        AND timestamp >= NOW() - INTERVAL '${minutes} minutes'
      ORDER BY timestamp ASC
    `;

    const currentTPM = data.length > 0
      ? Number(data[data.length - 1].tokens)
      : 0;

    return { 
      success: true, 
      data, 
      currentTPM,
      total: data.reduce((sum, r) => sum + Number(r.tokens), 0)
    };
  } catch (error) {
    console.error("Get TPM error:", error);
    return { success: false, data: [], currentTPM: 0, error: String(error) };
  }
}

// Get TPH data (last N hours)
export async function getTPHData(hours: number = 24) {
  try {
    const data = await sql`
      SELECT 
        hour_start as timestamp, total_requests as requests, 
        total_tokens as tokens, total_input_tokens as input_tokens,
        total_output_tokens as output_tokens, avg_latency_ms,
        success_count, failure_count
      FROM hourly_aggregates
      WHERE hour_start >= NOW() - INTERVAL '${hours} hours'
      ORDER BY hour_start ASC
    `;

    const currentTPH = data.length > 0
      ? Number(data[data.length - 1].tokens)
      : 0;

    return { 
      success: true, 
      data, 
      currentTPH,
      total: data.reduce((sum, r) => sum + Number(r.tokens), 0)
    };
  } catch (error) {
    console.error("Get TPH error:", error);
    return { success: false, data: [], currentTPH: 0, error: String(error) };
  }
}

// Get TPD data (last N days)
export async function getTPDData(days: number = 30) {
  try {
    const data = await sql`
      SELECT 
        date as timestamp, total_requests as requests, 
        total_tokens as tokens, total_input_tokens as input_tokens,
        total_output_tokens as output_tokens, avg_latency_ms,
        success_count, failure_count, peak_tps, peak_tpm
      FROM daily_aggregates
      WHERE date >= CURRENT_DATE - INTERVAL '${days} days'
      ORDER BY date ASC
    `;

    const currentTPD = data.length > 0
      ? Number(data[data.length - 1].tokens)
      : 0;

    return { 
      success: true, 
      data, 
      currentTPD,
      total: data.reduce((sum, r) => sum + Number(r.tokens), 0)
    };
  } catch (error) {
    console.error("Get TPD error:", error);
    return { success: false, data: [], currentTPD: 0, error: String(error) };
  }
}

// Update or insert daily aggregate
export async function upsertDailyAggregate(data: {
  date: Date;
  requests: number;
  tokens: number;
  inputTokens: number;
  outputTokens: number;
  successCount: number;
  failureCount: number;
  avgLatencyMs: number;
  peakTps?: number;
  peakTpm?: number;
}) {
  try {
    const dateStr = data.date.toISOString().split("T")[0];

    await sql`
      INSERT INTO daily_aggregates (
        date, total_requests, total_tokens, total_input_tokens, 
        total_output_tokens, success_count, failure_count, 
        avg_latency_ms, peak_tps, peak_tpm
      ) VALUES (
        ${dateStr}, ${data.requests}, ${data.tokens}, ${data.inputTokens},
        ${data.outputTokens}, ${data.successCount}, ${data.failureCount},
        ${data.avgLatencyMs}, ${data.peakTps || 0}, ${data.peakTpm || 0}
      )
      ON CONFLICT (date) DO UPDATE SET
        total_requests = daily_aggregates.total_requests + EXCLUDED.total_requests,
        total_tokens = daily_aggregates.total_tokens + EXCLUDED.total_tokens,
        total_input_tokens = daily_aggregates.total_input_tokens + EXCLUDED.total_input_tokens,
        total_output_tokens = daily_aggregates.total_output_tokens + EXCLUDED.total_output_tokens,
        success_count = daily_aggregates.success_count + EXCLUDED.success_count,
        failure_count = daily_aggregates.failure_count + EXCLUDED.failure_count,
        avg_latency_ms = (daily_aggregates.avg_latency_ms + EXCLUDED.avg_latency_ms) / 2,
        peak_tps = GREATEST(daily_aggregates.peak_tps, EXCLUDED.peak_tps),
        peak_tpm = GREATEST(daily_aggregates.peak_tpm, EXCLUDED.peak_tpm),
        updated_at = NOW()
    `;

    return { success: true };
  } catch (error) {
    console.error("Upsert daily aggregate error:", error);
    return { success: false, error: String(error) };
  }
}

// Update or insert hourly aggregate
export async function upsertHourlyAggregate(data: {
  hourStart: Date;
  requests: number;
  tokens: number;
  inputTokens: number;
  outputTokens: number;
  successCount: number;
  failureCount: number;
  avgLatencyMs: number;
}) {
  try {
    // Round to hour start
    const hourStart = new Date(data.hourStart);
    hourStart.setMinutes(0, 0, 0);

    await sql`
      INSERT INTO hourly_aggregates (
        hour_start, total_requests, total_tokens, total_input_tokens, 
        total_output_tokens, success_count, failure_count, avg_latency_ms
      ) VALUES (
        ${hourStart.toISOString()}, ${data.requests}, ${data.tokens}, ${data.inputTokens},
        ${data.outputTokens}, ${data.successCount}, ${data.failureCount}, ${data.avgLatencyMs}
      )
      ON CONFLICT (hour_start) DO UPDATE SET
        total_requests = hourly_aggregates.total_requests + EXCLUDED.total_requests,
        total_tokens = hourly_aggregates.total_tokens + EXCLUDED.total_tokens,
        total_input_tokens = hourly_aggregates.total_input_tokens + EXCLUDED.total_input_tokens,
        total_output_tokens = hourly_aggregates.total_output_tokens + EXCLUDED.total_output_tokens,
        success_count = hourly_aggregates.success_count + EXCLUDED.success_count,
        failure_count = hourly_aggregates.failure_count + EXCLUDED.failure_count,
        avg_latency_ms = (hourly_aggregates.avg_latency_ms + EXCLUDED.avg_latency_ms) / 2,
        updated_at = NOW()
    `;

    return { success: true };
  } catch (error) {
    console.error("Upsert hourly aggregate error:", error);
    return { success: false, error: String(error) };
  }
}

// Get summary statistics
export async function getSummaryStats(range: "1h" | "24h" | "7d" | "30d" = "24h") {
  try {
    let interval: string;
    switch (range) {
      case "1h": interval = "1 hour"; break;
      case "24h": interval = "24 hours"; break;
      case "7d": interval = "7 days"; break;
      case "30d": interval = "30 days"; break;
      default: interval = "24 hours";
    }

    const result = await sql`
      SELECT 
        COALESCE(SUM(requests), 0) as total_requests,
        COALESCE(SUM(tokens), 0) as total_tokens,
        COALESCE(SUM(success_count), 0) as success_count,
        COALESCE(SUM(failure_count), 0) as failure_count,
        COALESCE(AVG(avg_latency_ms), 0) as avg_latency_ms,
        COALESCE(MAX(requests), 0) as peak_requests,
        COALESCE(MAX(tokens), 0) as peak_tokens
      FROM metrics_snapshots
      WHERE timestamp >= NOW() - INTERVAL '${interval}'
    `;

    const stats = result[0];
    const totalRequests = Number(stats.total_requests);
    const successCount = Number(stats.success_count);
    const successRate = totalRequests > 0 ? (successCount / totalRequests) * 100 : 0;

    return {
      success: true,
      data: {
        totalRequests,
        totalTokens: Number(stats.total_tokens),
        successCount,
        failureCount: Number(stats.failure_count),
        successRate,
        avgLatencyMs: Number(stats.avg_latency_ms),
        peakTps: Number(stats.peak_requests),
        peakTpm: Number(stats.peak_tokens),
      },
    };
  } catch (error) {
    console.error("Get summary stats error:", error);
    return { success: false, data: null, error: String(error) };
  }
}

// Get model breakdown
export async function getModelBreakdown(range: "1h" | "24h" | "7d" | "30d" = "24h") {
  try {
    let interval: string;
    switch (range) {
      case "1h": interval = "1 hour"; break;
      case "24h": interval = "24 hours"; break;
      case "7d": interval = "7 days"; break;
      case "30d": interval = "30 days"; break;
      default: interval = "24 hours";
    }

    const data = await sql`
      SELECT 
        mm.model_name,
        SUM(mm.requests) as total_requests,
        SUM(mm.tokens) as total_tokens,
        SUM(mm.input_tokens) as input_tokens,
        SUM(mm.output_tokens) as output_tokens,
        AVG(mm.avg_latency_ms) as avg_latency_ms
      FROM model_metrics mm
      JOIN metrics_snapshots ms ON mm.snapshot_id = ms.id
      WHERE ms.timestamp >= NOW() - INTERVAL '${interval}'
      GROUP BY mm.model_name
      ORDER BY total_requests DESC
    `;

    return { success: true, data };
  } catch (error) {
    console.error("Get model breakdown error:", error);
    return { success: false, data: [], error: String(error) };
  }
}

// Cleanup old data (retention policy)
export async function cleanupOldData(retentionDays: number = 30) {
  try {
    // Delete second-granularity data older than 1 hour
    await sql`
      DELETE FROM metrics_snapshots 
      WHERE granularity = 'second' 
        AND timestamp < NOW() - INTERVAL '1 hour'
    `;

    // Delete minute-granularity data older than 24 hours
    await sql`
      DELETE FROM metrics_snapshots 
      WHERE granularity = 'minute' 
        AND timestamp < NOW() - INTERVAL '24 hours'
    `;

    // Delete hour-granularity data older than 7 days
    await sql`
      DELETE FROM metrics_snapshots 
      WHERE granularity = 'hour' 
        AND timestamp < NOW() - INTERVAL '7 days'
    `;

    // Delete daily data older than retention period
    await sql`
      DELETE FROM metrics_snapshots 
      WHERE granularity = 'day' 
        AND timestamp < NOW() - INTERVAL '${retentionDays} days'
    `;

    await sql`
      DELETE FROM hourly_aggregates 
      WHERE hour_start < NOW() - INTERVAL '${retentionDays} days'
    `;

    await sql`
      DELETE FROM daily_aggregates 
      WHERE date < CURRENT_DATE - INTERVAL '${retentionDays} days'
    `;

    return { success: true };
  } catch (error) {
    console.error("Cleanup error:", error);
    return { success: false, error: String(error) };
  }
}
