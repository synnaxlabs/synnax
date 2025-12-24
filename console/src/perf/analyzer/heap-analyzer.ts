// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { math } from "@synnaxlabs/x";

import {
  type LeakReport,
  type Severity,
  type Trend,
  ZERO_LEAK_REPORT,
} from "@/perf/analyzer/types";
import {
  HEAP_COMPARISON_WINDOW_SIZE,
  HEAP_SLOPE_THRESHOLD,
  THRESHOLDS,
} from "@/perf/constants";
import { type HeapSnapshot } from "@/perf/metrics/types";

/**
 * Analyzes heap snapshots to detect memory leaks.
 * Compares baseline (first N samples) vs recent (last N samples).
 * Severity is determined by growth percent with an increasing trend.
 */
export class HeapAnalyzer {
  analyze(snapshots: HeapSnapshot[]): LeakReport {
    if (snapshots.length < 2)
      return { ...ZERO_LEAK_REPORT, snapshotCount: snapshots.length };

    const windowSize = Math.min(
      HEAP_COMPARISON_WINDOW_SIZE,
      Math.floor(snapshots.length / 2),
    );
    const baseline = snapshots.slice(0, windowSize);
    const recent = snapshots.slice(-windowSize);

    const avgBaseline = math.average(baseline.map((s) => s.heapUsedMB));
    const avgRecent = math.average(recent.map((s) => s.heapUsedMB));

    const growthMB = avgRecent - avgBaseline;
    const growthPercent = avgBaseline > 0 ? (growthMB / avgBaseline) * 100 : 0;

    const trend = this.calculateTrend(snapshots);

    let severity: Severity = "none";
    if (trend === "increasing")
      if (growthPercent > THRESHOLDS.heapGrowth.error) severity = "error";
      else if (growthPercent > THRESHOLDS.heapGrowth.warn) severity = "warning";

    return {
      severity,
      heapStartMB: math.roundTo(avgBaseline),
      heapEndMB: math.roundTo(avgRecent),
      heapGrowthMB: math.roundTo(growthMB, 2),
      heapGrowthPercent: math.roundTo(growthPercent, 2),
      trend,
      snapshotCount: snapshots.length,
    };
  }

  private calculateTrend(snapshots: HeapSnapshot[]): Trend {
    if (snapshots.length < 2) return "stable";

    const n = snapshots.length;
    let sumX = 0,
      sumY = 0,
      sumXY = 0,
      sumX2 = 0;

    for (let i = 0; i < n; i++) {
      sumX += i;
      sumY += snapshots[i].heapUsedMB;
      sumXY += i * snapshots[i].heapUsedMB;
      sumX2 += i * i;
    }

    const denominator = n * sumX2 - sumX * sumX;
    if (denominator === 0) return "stable";

    const slope = (n * sumXY - sumX * sumY) / denominator;
    const avgHeap = sumY / n;
    const normalizedSlope = avgHeap > 0 ? slope / avgHeap : slope;

    if (normalizedSlope > HEAP_SLOPE_THRESHOLD) return "increasing";
    if (normalizedSlope < -HEAP_SLOPE_THRESHOLD) return "decreasing";
    return "stable";
  }
}
