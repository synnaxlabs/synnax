// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useRef } from "react";

import { CpuAnalyzer } from "@/perf/analyzer/cpu-analyzer";
import { FpsAnalyzer } from "@/perf/analyzer/fps-analyzer";
import { GpuAnalyzer } from "@/perf/analyzer/gpu-analyzer";
import { HeapAnalyzer } from "@/perf/analyzer/heap-analyzer";
import {
  type CpuReport,
  type FpsReport,
  type GpuReport,
  type LeakReport,
} from "@/perf/analyzer/types";
import { type CapturedValues } from "@/perf/hooks/useCapturedValues";
import { type Aggregates } from "@/perf/metrics/buffer";
import { type MetricSample } from "@/perf/metrics/types";

export interface AnalyzeInput {
  samples: MetricSample[];
  currentSample: MetricSample;
  captured: CapturedValues;
  aggregates: Aggregates;
}

export interface AnalysisResults {
  leak: LeakReport;
  fps: FpsReport;
  cpu: CpuReport;
  gpu: GpuReport;
}

export interface UseProfilingAnalyzersResult {
  analyze: (input: AnalyzeInput) => AnalysisResults;
}

/**
 * Hook that manages analyzer instances and runs analysis on profiling data.
 *
 * Encapsulates:
 * - Analyzer instantiation (LeakDetector, FpsAnalyzer, CpuAnalyzer, GpuAnalyzer)
 * - Running analysis with proper input transformation
 */
export const useProfilingAnalyzers = (): UseProfilingAnalyzersResult => {
  const analyzersRef = useRef({
    heap: new HeapAnalyzer(),
    fps: new FpsAnalyzer(),
    cpu: new CpuAnalyzer(),
    gpu: new GpuAnalyzer(),
  });

  const analyze = useCallback((input: AnalyzeInput): AnalysisResults => {
    const { samples, currentSample, captured, aggregates } = input;
    const analyzers = analyzersRef.current;

    const heapSnapshots = samples
      .filter((s) => s.heapUsedMB != null)
      .map((s) => ({
        timestamp: s.timestamp,
        heapUsedMB: s.heapUsedMB!,
        heapTotalMB: s.heapTotalMB!,
      }));
    const leak = analyzers.heap.analyze(heapSnapshots);

    const fps = analyzers.fps.analyze({
      startFps: captured.initialFPS,
      endFps: currentSample.frameRate,
      minFps: aggregates.minFps,
      avgFps: aggregates.avgFps,
    });

    const cpu = analyzers.cpu.analyze({
      startPercent: captured.initialCPU,
      endPercent: currentSample.cpuPercent,
      avgPercent: aggregates.avgCpu,
      maxPercent: aggregates.maxCpu,
    });

    const gpu = analyzers.gpu.analyze({
      startPercent: captured.initialGPU,
      endPercent: currentSample.gpuPercent,
      avgPercent: aggregates.avgGpu,
      maxPercent: aggregates.maxGpu,
    });

    return { leak, fps, cpu, gpu };
  }, []);

  return { analyze };
};
