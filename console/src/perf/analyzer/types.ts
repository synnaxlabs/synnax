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

export const COMPARISON_WINDOW_SIZE = 30;

export type Trend = "increasing" | "stable" | "decreasing";

/** Report on memory leak detection. */
export interface LeakReport {
  detected: boolean;
  heapStartMB: number;
  heapEndMB: number;
  heapGrowthMB: number;
  heapGrowthPercent: number;
  trend: Trend;
  snapshotCount: number;
}

export interface DegradationReport {
  detected: boolean;
  averageFrameRateStart: number;
  averageFrameRateEnd: number;
  frameRateDegradationPercent: number;
}

export interface CpuReport {
  detected: boolean;
  avgPercent: number | null;
  peakPercent: number | null;
  startPercent: number | null;
  endPercent: number | null;
}

export interface GpuReport {
  detected: boolean;
  avgPercent: number | null;
  peakPercent: number | null;
  startPercent: number | null;
  endPercent: number | null;
}

/** Complete performance report. */
export interface PerfReport {
  durationMs: number;
  totalSamples: number;
  averageFrameRate: number;
  minFrameRate: number;
  maxFrameRate: number;
  averageHeapUsedMB: number | null;
  peakHeapUsedMB: number | null;
  totalNetworkRequests: number;
  leakReport: LeakReport;
  degradationReport: DegradationReport;
  cpuReport: CpuReport;
  gpuReport: GpuReport;
  workflowResults: WorkflowResult[];
}

export const ZERO_LEAK_REPORT: LeakReport = {
  detected: false,
  heapStartMB: 0,
  heapEndMB: 0,
  heapGrowthMB: 0,
  heapGrowthPercent: 0,
  trend: "stable",
  snapshotCount: 0,
};

export const ZERO_DEGRADATION_REPORT: DegradationReport = {
  detected: false,
  averageFrameRateStart: 0,
  averageFrameRateEnd: 0,
  frameRateDegradationPercent: 0,
};

export const ZERO_CPU_REPORT: CpuReport = {
  detected: false,
  avgPercent: null,
  peakPercent: null,
  startPercent: null,
  endPercent: null,
};

export const ZERO_GPU_REPORT: GpuReport = {
  detected: false,
  avgPercent: null,
  peakPercent: null,
  startPercent: null,
  endPercent: null,
};

/** Generate a performance report from collected data (stub for future export). */
export const generateReport = (
  samples: MetricSample[],
  _heapSnapshots: HeapSnapshot[],
  workflowResults: WorkflowResult[],
  startTime: number,
  endTime: number,
  leakReport: LeakReport,
  degradationReport: DegradationReport,
  cpuReport: CpuReport,
  gpuReport: GpuReport,
): PerfReport => {
  const frameRates = samples.map((s) => s.frameRate);
  const heapValues = samples
    .map((s) => s.heapUsedMB)
    .filter((h): h is number => h != null);

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
    cpuReport,
    gpuReport,
    workflowResults,
  };
};
