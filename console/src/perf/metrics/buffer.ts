// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { math } from "@synnaxlabs/x";

import { type MetricSample } from "@/perf/metrics/types";

const ZERO_SAMPLE: MetricSample = {
  timestamp: 0,
  cpuPercent: null,
  gpuPercent: null,
  heapUsedMB: null,
  heapTotalMB: null,
  frameRate: 0,
  longTaskCount: 0,
  longTaskDurationMs: 0,
  networkRequestCount: 0,
};

/** Pre-allocated buffer: baseline (first N) + recent (circular, last N). */
export class SampleBuffer {
  private readonly baseline: MetricSample[];
  private readonly recent: MetricSample[];
  private readonly baselineSize: number;
  private readonly recentSize: number;
  private baselineCount = 0;
  private recentIndex = 0;
  private recentCount = 0;
  private totalPushCount = 0;

  constructor(baselineSize = 60, recentSize = 60) {
    this.baselineSize = baselineSize;
    this.recentSize = recentSize;
    this.baseline = Array.from({ length: baselineSize }, () => ({ ...ZERO_SAMPLE }));
    this.recent = Array.from({ length: recentSize }, () => ({ ...ZERO_SAMPLE }));
  }

  push(sample: MetricSample): void {
    this.totalPushCount++;

    // Fill baseline first, then switch to recent (no overlap)
    if (this.baselineCount < this.baselineSize) {
      this.baseline[this.baselineCount] = sample;
      this.baselineCount++;
    } else {
      this.recent[this.recentIndex] = sample;
      this.recentIndex = (this.recentIndex + 1) % this.recentSize;
      if (this.recentCount < this.recentSize) this.recentCount++;
    }
  }

  getBaselineSamples(): MetricSample[] {
    return this.baseline.slice(0, this.baselineCount);
  }

  getRecentSamples(): MetricSample[] {
    if (this.recentCount === 0) return [];
    if (this.recentCount < this.recentSize)
      return this.recent.slice(0, this.recentCount);
    return [
      ...this.recent.slice(this.recentIndex),
      ...this.recent.slice(0, this.recentIndex),
    ];
  }

  getAllSamples(): MetricSample[] {
    return [...this.getBaselineSamples(), ...this.getRecentSamples()];
  }

  getTotalSampleCount(): number {
    return this.totalPushCount;
  }

  /** Compute aggregates on demand from current samples. */
  getAggregates(): {
    avgCpu: number | null;
    peakCpu: number | null;
    avgGpu: number | null;
    peakGpu: number | null;
    avgHeap: number | null;
  } {
    const samples = this.getAllSamples();
    const cpuValues = samples
      .map((s) => s.cpuPercent)
      .filter((v): v is number => v != null);
    const gpuValues = samples
      .map((s) => s.gpuPercent)
      .filter((v): v is number => v != null);
    const heapValues = samples
      .map((s) => s.heapUsedMB)
      .filter((v): v is number => v != null);

    return {
      avgCpu: cpuValues.length > 0 ? math.average(cpuValues) : null,
      peakCpu: cpuValues.length > 0 ? Math.max(...cpuValues) : null,
      avgGpu: gpuValues.length > 0 ? math.average(gpuValues) : null,
      peakGpu: gpuValues.length > 0 ? Math.max(...gpuValues) : null,
      avgHeap: heapValues.length > 0 ? math.average(heapValues) : null,
    };
  }

  reset(): void {
    this.baselineCount = 0;
    this.recentIndex = 0;
    this.recentCount = 0;
    this.totalPushCount = 0;
  }
}
