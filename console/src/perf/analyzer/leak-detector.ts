// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { math } from "@synnaxlabs/x";

import { type LeakReport, type Trend, ZERO_LEAK_REPORT } from "@/perf/analyzer/types";
import { type HeapSnapshot } from "@/perf/metrics/types";

/** Threshold for heap growth percentage to consider as a leak. */
const LEAK_THRESHOLD_PERCENT = 20;

/** Threshold for slope to determine trend direction. */
const SLOPE_THRESHOLD = 0.1;

/**
 * Analyzes heap snapshots to detect memory leaks.
 * Uses linear regression to determine if heap usage is trending upward.
 */
export class LeakDetector {
  /**
   * Analyze heap snapshots for memory leaks.
   * @param snapshots Array of heap snapshots over time.
   * @returns LeakReport with analysis results.
   */
  analyze(snapshots: HeapSnapshot[]): LeakReport {
    if (snapshots.length < 2) 
      return {
        ...ZERO_LEAK_REPORT,
        snapshotCount: snapshots.length,
      };
    

    const { first: avgFirst, last: avgLast } = math.compareQuarters(
      snapshots,
      (s) => s.heapUsedMB,
    );

    const growthMB = avgLast - avgFirst;
    const growthPercent = avgFirst > 0 ? ((avgLast - avgFirst) / avgFirst) * 100 : 0;

    // Determine trend using linear regression
    const trend = this.calculateTrend(snapshots);

    // Leak detected if heap grew by more than threshold and trend is increasing
    const detected = growthPercent > LEAK_THRESHOLD_PERCENT && trend === "increasing";

    return {
      detected,
      heapStartMB: math.roundTo(avgFirst),
      heapEndMB: math.roundTo(avgLast),
      heapGrowthMB: math.roundTo(growthMB, 2),
      heapGrowthPercent: math.roundTo(growthPercent, 2),
      trend,
      snapshotCount: snapshots.length,
    };
  }

  /**
   * Calculate the trend of heap usage using linear regression.
   */
  private calculateTrend(snapshots: HeapSnapshot[]): Trend {
    if (snapshots.length < 2) return "stable";

    // Simple linear regression to calculate slope
    const n = snapshots.length;
    let sumX = 0;
    let sumY = 0;
    let sumXY = 0;
    let sumX2 = 0;

    snapshots.forEach((s, i) => {
      sumX += i;
      sumY += s.heapUsedMB;
      sumXY += i * s.heapUsedMB;
      sumX2 += i * i;
    });

    const denominator = n * sumX2 - sumX * sumX;
    if (denominator === 0) return "stable";

    const slope = (n * sumXY - sumX * sumY) / denominator;

    // Normalize slope by the average heap size to make threshold meaningful
    const avgHeap = sumY / n;
    const normalizedSlope = avgHeap > 0 ? slope / avgHeap : slope;

    if (normalizedSlope > SLOPE_THRESHOLD) return "increasing";
    if (normalizedSlope < -SLOPE_THRESHOLD) return "decreasing";
    return "stable";
  }

}
