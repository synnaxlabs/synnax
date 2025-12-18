// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { WARMUP_SAMPLES } from "@/perf/constants";
import { type MetricSample } from "@/perf/metrics/types";

const ZERO_SAMPLE: MetricSample = {
  timestamp: 0,
  cpuPercent: null,
  gpuPercent: null,
  heapUsedMB: null,
  heapTotalMB: null,
  frameRate: null,
  longTaskCount: 0,
  longTaskDurationMs: 0,
  networkRequestCount: 0,
};

export interface Aggregates {
  avgFps: number | null;
  peakFps: number | null;
  minFps: number | null;
  avgCpu: number | null;
  peakCpu: number | null;
  avgGpu: number | null;
  peakGpu: number | null;
  avgHeap: number | null;
  peakHeap: number | null;
}

export const ZERO_AGGREGATES: Aggregates = {
  avgFps: null,
  peakFps: null,
  minFps: null,
  avgCpu: null,
  peakCpu: null,
  avgGpu: null,
  peakGpu: null,
  avgHeap: null,
  peakHeap: null,
};

interface RunningAggregate {
  avg: number;
  min: number;
  max: number;
  count: number;
}

const ZERO_AGGREGATE: RunningAggregate = { avg: 0, min: Infinity, max: -Infinity, count: 0 };

const updateAggregate = (agg: RunningAggregate, value: number, skipMax = false): void => {
  agg.count++;
  agg.avg += (value - agg.avg) / agg.count;
  agg.min = Math.min(agg.min, value);
  if (!skipMax) {
    agg.max = Math.max(agg.max, value);
  }
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
  private fpsAgg: RunningAggregate = { ...ZERO_AGGREGATE };
  private cpuAgg: RunningAggregate = { ...ZERO_AGGREGATE };
  private gpuAgg: RunningAggregate = { ...ZERO_AGGREGATE };
  private heapAgg: RunningAggregate = { ...ZERO_AGGREGATE };

  constructor(baselineSize = 60, recentSize = 60) {
    this.baselineSize = baselineSize;
    this.recentSize = recentSize;
    this.baseline = Array.from({ length: baselineSize }, () => ({ ...ZERO_SAMPLE }));
    this.recent = Array.from({ length: recentSize }, () => ({ ...ZERO_SAMPLE }));
  }

  push(sample: MetricSample): void {
    this.totalPushCount++;
    const inWarmup = this.totalPushCount <= WARMUP_SAMPLES;

    if (sample.frameRate != null) updateAggregate(this.fpsAgg, sample.frameRate, inWarmup);
    if (sample.cpuPercent != null) updateAggregate(this.cpuAgg, sample.cpuPercent, inWarmup);
    if (sample.gpuPercent != null) updateAggregate(this.gpuAgg, sample.gpuPercent, inWarmup);
    if (sample.heapUsedMB != null) updateAggregate(this.heapAgg, sample.heapUsedMB);

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

  /** Get true total averages computed via running aggregates (O(1) memory). */
  getAggregates(): Aggregates {
    const getMax = (agg: RunningAggregate) =>
      agg.count > 0 && agg.max !== -Infinity ? agg.max : null;
    const getMin = (agg: RunningAggregate) =>
      agg.count > 0 && agg.min !== Infinity ? agg.min : null;
    const getAvg = (agg: RunningAggregate) =>
      agg.count > 0 ? agg.avg : null;

    return {
      avgFps: getAvg(this.fpsAgg),
      peakFps: getMax(this.fpsAgg),
      minFps: getMin(this.fpsAgg),
      avgCpu: getAvg(this.cpuAgg),
      peakCpu: getMax(this.cpuAgg),
      avgGpu: getAvg(this.gpuAgg),
      peakGpu: getMax(this.gpuAgg),
      avgHeap: getAvg(this.heapAgg),
      peakHeap: getMax(this.heapAgg),
    };
  }

  reset(): void {
    this.baselineCount = 0;
    this.recentIndex = 0;
    this.recentCount = 0;
    this.totalPushCount = 0;
    this.fpsAgg = { ...ZERO_AGGREGATE };
    this.cpuAgg = { ...ZERO_AGGREGATE };
    this.gpuAgg = { ...ZERO_AGGREGATE };
    this.heapAgg = { ...ZERO_AGGREGATE };
  }
}
