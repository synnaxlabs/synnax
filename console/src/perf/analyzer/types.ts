// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type HeapSnapshot, type MetricSample } from "@/perf/metrics/types";
import { type WorkflowResult } from "@/perf/workflows/types";

/** Trend direction for metrics over time. */
export type Trend = "increasing" | "stable" | "decreasing";

/** Report on memory leak detection. */
export interface LeakReport {
  /** Whether a memory leak was detected. */
  detected: boolean;
  /** Heap used at start of test in MB. */
  heapStartMB: number;
  /** Heap used at end of test in MB. */
  heapEndMB: number;
  /** Heap growth in MB from start to end. */
  heapGrowthMB: number;
  /** Heap growth as a percentage. */
  heapGrowthPercent: number;
  /** Trend of heap usage over time. */
  trend: Trend;
  /** Number of heap snapshots analyzed. */
  snapshotCount: number;
}

/** Report on performance degradation. */
export interface DegradationReport {
  /** Whether performance degradation was detected. */
  detected: boolean;
  /** Average frame rate in the first quarter of samples. */
  averageFrameRateStart: number;
  /** Average frame rate in the last quarter of samples. */
  averageFrameRateEnd: number;
  /** Percentage drop in frame rate. */
  frameRateDegradationPercent: number;
  /** Trend of long task occurrences. */
  longTaskTrend: Trend;
  /** Total long task count. */
  totalLongTasks: number;
  /** Total long task duration in ms. */
  totalLongTaskDurationMs: number;
}

/** Complete performance report. */
export interface PerfReport {
  /** Total duration of the harness run in ms. */
  durationMs: number;
  /** Total number of metric samples collected. */
  totalSamples: number;
  /** Average frame rate across all samples. */
  averageFrameRate: number;
  /** Minimum frame rate observed. */
  minFrameRate: number;
  /** Maximum frame rate observed. */
  maxFrameRate: number;
  /** Average heap used in MB (null if not available). */
  averageHeapUsedMB: number | null;
  /** Peak heap used in MB (null if not available). */
  peakHeapUsedMB: number | null;
  /** Total network requests made. */
  totalNetworkRequests: number;
  /** Memory leak analysis. */
  leakReport: LeakReport;
  /** Performance degradation analysis. */
  degradationReport: DegradationReport;
  /** Results from workflow executions. */
  workflowResults: WorkflowResult[];
}

/** Default empty leak report. */
export const ZERO_LEAK_REPORT: LeakReport = {
  detected: false,
  heapStartMB: 0,
  heapEndMB: 0,
  heapGrowthMB: 0,
  heapGrowthPercent: 0,
  trend: "stable",
  snapshotCount: 0,
};

/** Default empty degradation report. */
export const ZERO_DEGRADATION_REPORT: DegradationReport = {
  detected: false,
  averageFrameRateStart: 0,
  averageFrameRateEnd: 0,
  frameRateDegradationPercent: 0,
  longTaskTrend: "stable",
  totalLongTasks: 0,
  totalLongTaskDurationMs: 0,
};

/** Generate a performance report from collected data. */
export const generateReport = (
  samples: MetricSample[],
  _heapSnapshots: HeapSnapshot[],
  workflowResults: WorkflowResult[],
  startTime: number,
  endTime: number,
  leakReport: LeakReport,
  degradationReport: DegradationReport,
): PerfReport => {
  const frameRates = samples.map((s) => s.frameRate);
  const heapValues = samples.map((s) => s.heapUsedMB).filter((h): h is number => h != null);

  return {
    durationMs: endTime - startTime,
    totalSamples: samples.length,
    averageFrameRate:
      frameRates.length > 0
        ? frameRates.reduce((a, b) => a + b, 0) / frameRates.length
        : 0,
    minFrameRate: frameRates.length > 0 ? Math.min(...frameRates) : 0,
    maxFrameRate: frameRates.length > 0 ? Math.max(...frameRates) : 0,
    averageHeapUsedMB:
      heapValues.length > 0
        ? heapValues.reduce((a, b) => a + b, 0) / heapValues.length
        : null,
    peakHeapUsedMB: heapValues.length > 0 ? Math.max(...heapValues) : null,
    totalNetworkRequests: samples.reduce((sum, s) => sum + s.networkRequestCount, 0),
    leakReport,
    degradationReport,
    workflowResults,
  };
};
