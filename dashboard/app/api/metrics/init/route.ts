import { NextResponse } from "next/server";
import { neon } from "@neondatabase/serverless";

const sql = neon(process.env.DATABASE_URL!);

// POST - Initialize database tables
export async function POST() {
  try {
    // Metrics snapshots table
    await sql`
      CREATE TABLE IF NOT EXISTS metrics_snapshots (
        id SERIAL PRIMARY KEY,
        timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        granularity VARCHAR(20) NOT NULL,
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

    await sql`
      CREATE INDEX IF NOT EXISTS idx_metrics_snapshots_timestamp 
      ON metrics_snapshots(timestamp DESC)
    `;

    await sql`
      CREATE INDEX IF NOT EXISTS idx_metrics_snapshots_granularity 
      ON metrics_snapshots(granularity, timestamp DESC)
    `;

    // Model metrics table
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

    // Daily aggregates table
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

    return NextResponse.json({ 
      success: true, 
      message: "Database initialized successfully",
      tables: [
        "metrics_snapshots",
        "model_metrics", 
        "daily_aggregates",
        "hourly_aggregates"
      ]
    });
  } catch (error) {
    console.error("Database init error:", error);
    return NextResponse.json(
      { success: false, error: String(error) },
      { status: 500 }
    );
  }
}

// GET - Check database status
export async function GET() {
  try {
    // Check if tables exist
    const tables = await sql`
      SELECT table_name 
      FROM information_schema.tables 
      WHERE table_schema = 'public' 
        AND table_name IN ('metrics_snapshots', 'model_metrics', 'daily_aggregates', 'hourly_aggregates')
    `;

    const existingTables = tables.map(t => t.table_name);

    // Get row counts
    let counts: Record<string, number> = {};
    
    if (existingTables.includes("metrics_snapshots")) {
      const result = await sql`SELECT COUNT(*) as count FROM metrics_snapshots`;
      counts.metrics_snapshots = Number(result[0].count);
    }
    
    if (existingTables.includes("model_metrics")) {
      const result = await sql`SELECT COUNT(*) as count FROM model_metrics`;
      counts.model_metrics = Number(result[0].count);
    }
    
    if (existingTables.includes("daily_aggregates")) {
      const result = await sql`SELECT COUNT(*) as count FROM daily_aggregates`;
      counts.daily_aggregates = Number(result[0].count);
    }
    
    if (existingTables.includes("hourly_aggregates")) {
      const result = await sql`SELECT COUNT(*) as count FROM hourly_aggregates`;
      counts.hourly_aggregates = Number(result[0].count);
    }

    const allTablesExist = existingTables.length === 4;

    return NextResponse.json({
      success: true,
      initialized: allTablesExist,
      tables: existingTables,
      counts,
      missingTables: ["metrics_snapshots", "model_metrics", "daily_aggregates", "hourly_aggregates"]
        .filter(t => !existingTables.includes(t))
    });
  } catch (error) {
    console.error("Database status error:", error);
    return NextResponse.json(
      { success: false, error: String(error), initialized: false },
      { status: 500 }
    );
  }
}
