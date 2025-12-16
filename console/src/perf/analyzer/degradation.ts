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
  type DegradationReport,
  type Trend,
  ZERO_DEGRADATION_REPORT,
} from "@/perf/analyzer/types";
import { type MetricSample } from "@/perf/metrics/types";

/** Threshold for FPS drop percentage to consider as degradation. */
const FPS_DEGRADATION_THRESHOLD_PERCENT = 15;

/**
 * Analyzes metric samples to detect performance degradation.
 * Compares frame rates at the start vs end of the test run.
 */
export class DegradationDetector {
  /**
   * Analyze samples for performance degradation.
   * @param samples Array of metric samples over time.
   * @returns DegradationReport with analysis results.
   */
  analyze(samples: MetricSample[]): DegradationReport {
    if (samples.length < 2) 
      return ZERO_DEGRADATION_REPORT;
    

    const { first: avgFPSStart, last: avgFPSEnd } = math.compareQuarters(
      samples,
      (s) => s.frameRate,
    );

    const fpsDrop = avgFPSStart > 0 ? ((avgFPSStart - avgFPSEnd) / avgFPSStart) * 100 : 0;

    // Calculate long task totals and trend
    const totalLongTasks = samples.reduce((sum, s) => sum + s.longTaskCount, 0);
    const totalLongTaskDurationMs = samples.reduce(
      (sum, s) => sum + s.longTaskDurationMs,
      0,
    );
    const longTaskTrend = this.calculateLongTaskTrend(samples);

    // Degradation detected if FPS dropped by more than threshold
    const detected = fpsDrop > FPS_DEGRADATION_THRESHOLD_PERCENT;

    return {
      detected,
      averageFrameRateStart: math.roundTo(avgFPSStart),
      averageFrameRateEnd: math.roundTo(avgFPSEnd),
      frameRateDegradationPercent: math.roundTo(fpsDrop, 2),
      longTaskTrend,
      totalLongTasks,
      totalLongTaskDurationMs: Math.round(totalLongTaskDurationMs),
    };
  }

  /**
   * Calculate the trend of long task occurrences.
   */
  private calculateLongTaskTrend(samples: MetricSample[]): Trend {
    if (samples.length < 4) return "stable";

    const { first: avgFirst, last: avgLast } = math.compareQuarters(
      samples,
      (s) => s.longTaskCount,
    );

    // Use a threshold relative to the average
    const threshold = Math.max(0.5, (avgFirst + avgLast) / 4);

    if (avgLast > avgFirst + threshold) return "increasing";
    if (avgLast < avgFirst - threshold) return "decreasing";
    return "stable";
  }
}
