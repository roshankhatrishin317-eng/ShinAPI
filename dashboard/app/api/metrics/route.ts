import { NextRequest, NextResponse } from "next/server";
import { neon } from "@neondatabase/serverless";

const sql = neon(process.env.DATABASE_URL!);

// GET - Retrieve metrics based on type and range
export async function GET(request: NextRequest) {
  try {
    const searchParams = request.nextUrl.searchParams;
    const type = searchParams.get("type") || "tps";
    const range = searchParams.get("range") || "1h";
    const granularity = searchParams.get("granularity") || "second";

    let data;
    let currentValue = 0;

    switch (type) {
      case "tps":
        data = await sql`
          SELECT 
            timestamp, requests, tokens, avg_latency_ms,
            success_count, failure_count
          FROM metrics_snapshots
          WHERE granularity = 'second'
          ORDER BY timestamp DESC
          LIMIT 60
        `;
        data = data.reverse();
        const recent = data.slice(-10);
        currentValue = recent.length > 0
          ? recent.reduce((sum, r) => sum + Number(r.requests), 0) / recent.length
          : 0;
        break;

      case "tpm":
        data = await sql`
          SELECT 
            timestamp, requests, tokens, input_tokens, output_tokens,
            avg_latency_ms, success_count, failure_count
          FROM metrics_snapshots
          WHERE granularity = 'minute'
          ORDER BY timestamp DESC
          LIMIT 60
        `;
        data = data.reverse();
        currentValue = data.length > 0 ? Number(data[data.length - 1].tokens) : 0;
        break;

      case "tph":
        data = await sql`
          SELECT 
            hour_start as timestamp, total_requests as requests, 
            total_tokens as tokens, total_input_tokens as input_tokens,
            total_output_tokens as output_tokens, avg_latency_ms,
            success_count, failure_count
          FROM hourly_aggregates
          ORDER BY hour_start DESC
          LIMIT 24
        `;
        data = data.reverse();
        currentValue = data.length > 0 ? Number(data[data.length - 1].tokens) : 0;
        break;

      case "tpd":
        data = await sql`
          SELECT 
            date as timestamp, total_requests as requests, 
            total_tokens as tokens, total_input_tokens as input_tokens,
            total_output_tokens as output_tokens, avg_latency_ms,
            success_count, failure_count, peak_tps, peak_tpm
          FROM daily_aggregates
          ORDER BY date DESC
          LIMIT 30
        `;
        data = data.reverse();
        currentValue = data.length > 0 ? Number(data[data.length - 1].tokens) : 0;
        break;

      case "summary":
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
          WHERE timestamp >= NOW() - INTERVAL '1 day' * ${range === "30d" ? 30 : range === "7d" ? 7 : range === "24h" ? 1 : 0.04}
        `;

        const stats = result[0];
        const totalRequests = Number(stats.total_requests);
        const successCount = Number(stats.success_count);

        return NextResponse.json({
          success: true,
          data: {
            total_requests: totalRequests,
            total_tokens: Number(stats.total_tokens),
            success_rate: totalRequests > 0 ? (successCount / totalRequests) * 100 : 0,
            avg_latency_ms: Number(stats.avg_latency_ms),
            peak_tps: Number(stats.peak_requests),
            peak_tpm: Number(stats.peak_tokens),
          },
        });

      default:
        return NextResponse.json({ error: "Invalid type" }, { status: 400 });
    }

    return NextResponse.json({
      success: true,
      type,
      [`current_${type}`]: currentValue,
      data,
    });
  } catch (error) {
    console.error("Metrics GET error:", error);
    return NextResponse.json(
      { success: false, error: String(error) },
      { status: 500 }
    );
  }
}

// POST - Insert new metrics data (called by Go backend)
export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    const { 
      granularity = "second",
      timestamp,
      requests = 0,
      tokens = 0,
      input_tokens = 0,
      output_tokens = 0,
      success_count = 0,
      failure_count = 0,
      avg_latency_ms = 0,
      model_metrics = [],
    } = body;

    // Validate granularity
    if (!["second", "minute", "hour", "day"].includes(granularity)) {
      return NextResponse.json(
        { error: "Invalid granularity" },
        { status: 400 }
      );
    }

    const ts = timestamp ? new Date(timestamp) : new Date();

    // Insert main snapshot
    const result = await sql`
      INSERT INTO metrics_snapshots (
        timestamp, granularity, requests, tokens, input_tokens, output_tokens,
        success_count, failure_count, avg_latency_ms
      ) VALUES (
        ${ts.toISOString()}, ${granularity}, ${requests}, ${tokens},
        ${input_tokens}, ${output_tokens}, ${success_count}, 
        ${failure_count}, ${avg_latency_ms}
      )
      RETURNING id
    `;

    const snapshotId = result[0]?.id;

    // Insert model metrics if provided
    if (snapshotId && model_metrics && model_metrics.length > 0) {
      for (const model of model_metrics) {
        await sql`
          INSERT INTO model_metrics (
            snapshot_id, model_name, requests, tokens, input_tokens, 
            output_tokens, avg_latency_ms
          ) VALUES (
            ${snapshotId}, ${model.model_name || model.modelName}, 
            ${model.requests || 0}, ${model.tokens || 0},
            ${model.input_tokens || model.inputTokens || 0}, 
            ${model.output_tokens || model.outputTokens || 0}, 
            ${model.avg_latency_ms || model.avgLatencyMs || 0}
          )
        `;
      }
    }

    // Update aggregates for minute/hour/day granularity
    if (granularity === "minute" || granularity === "hour") {
      // Update hourly aggregate
      const hourStart = new Date(ts);
      hourStart.setMinutes(0, 0, 0);

      await sql`
        INSERT INTO hourly_aggregates (
          hour_start, total_requests, total_tokens, total_input_tokens, 
          total_output_tokens, success_count, failure_count, avg_latency_ms
        ) VALUES (
          ${hourStart.toISOString()}, ${requests}, ${tokens}, ${input_tokens},
          ${output_tokens}, ${success_count}, ${failure_count}, ${avg_latency_ms}
        )
        ON CONFLICT (hour_start) DO UPDATE SET
          total_requests = hourly_aggregates.total_requests + EXCLUDED.total_requests,
          total_tokens = hourly_aggregates.total_tokens + EXCLUDED.total_tokens,
          total_input_tokens = hourly_aggregates.total_input_tokens + EXCLUDED.total_input_tokens,
          total_output_tokens = hourly_aggregates.total_output_tokens + EXCLUDED.total_output_tokens,
          success_count = hourly_aggregates.success_count + EXCLUDED.success_count,
          failure_count = hourly_aggregates.failure_count + EXCLUDED.failure_count,
          avg_latency_ms = (hourly_aggregates.avg_latency_ms * 0.9 + EXCLUDED.avg_latency_ms * 0.1),
          updated_at = NOW()
      `;
    }

    if (granularity === "hour" || granularity === "day") {
      // Update daily aggregate
      const dateStr = ts.toISOString().split("T")[0];

      await sql`
        INSERT INTO daily_aggregates (
          date, total_requests, total_tokens, total_input_tokens, 
          total_output_tokens, success_count, failure_count, avg_latency_ms
        ) VALUES (
          ${dateStr}, ${requests}, ${tokens}, ${input_tokens},
          ${output_tokens}, ${success_count}, ${failure_count}, ${avg_latency_ms}
        )
        ON CONFLICT (date) DO UPDATE SET
          total_requests = daily_aggregates.total_requests + EXCLUDED.total_requests,
          total_tokens = daily_aggregates.total_tokens + EXCLUDED.total_tokens,
          total_input_tokens = daily_aggregates.total_input_tokens + EXCLUDED.total_input_tokens,
          total_output_tokens = daily_aggregates.total_output_tokens + EXCLUDED.total_output_tokens,
          success_count = daily_aggregates.success_count + EXCLUDED.success_count,
          failure_count = daily_aggregates.failure_count + EXCLUDED.failure_count,
          avg_latency_ms = (daily_aggregates.avg_latency_ms * 0.9 + EXCLUDED.avg_latency_ms * 0.1),
          updated_at = NOW()
      `;
    }

    return NextResponse.json({ success: true, id: snapshotId });
  } catch (error) {
    console.error("Metrics POST error:", error);
    return NextResponse.json(
      { success: false, error: String(error) },
      { status: 500 }
    );
  }
}
