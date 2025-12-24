// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/client";
import { math } from "@synnaxlabs/x";
import { useCallback, useEffect, useRef } from "react";
import { useDispatch } from "react-redux";

import {
  ZERO_CPU_REPORT,
  ZERO_FPS_REPORT,
  ZERO_GPU_REPORT,
  ZERO_LEAK_REPORT,
} from "@/perf/analyzer/types";
import { useCapturedValues } from "@/perf/hooks/useCapturedValues";
import { type CollectorsState } from "@/perf/hooks/useCollectors";
import { useProfilingAnalyzers } from "@/perf/hooks/useProfilingAnalyzers";
import { type LiveValues, useProfilingRange } from "@/perf/hooks/useProfilingRange";
import { type Aggregates, type SampleBuffer } from "@/perf/metrics/buffer";
import { type MetricSample } from "@/perf/metrics/types";
import * as Perf from "@/perf/slice";
import { type HarnessStatus } from "@/perf/slice";

const getCollectorValues = (c: CollectorsState | null): LiveValues => ({
  fps: c?.fps?.getCurrentFPS() ?? null,
  cpu: c?.cpu?.getCpuPercent() ?? null,
  gpu: c?.gpu?.getGpuPercent() ?? null,
  heap: c?.heap?.getHeapUsedMB() ?? null,
});

export interface UseProfilingSessionOptions {
  status: HarnessStatus;
  collectors: React.RefObject<CollectorsState>;
  sampleBuffer: React.RefObject<SampleBuffer>;
  getAggregates: () => Aggregates;
  resetEventCollectors: () => void;
  resetTableData: () => void;
  resetBuffer: () => void;
}

export interface UseProfilingSessionResult {
  /** Callback to pass to useCollectors onSample. */
  handleSample: (sample: MetricSample, buffer: SampleBuffer) => void;
}

/**
 * Orchestrator hook that manages the entire profiling session lifecycle.
 *
 * Composes:
 * - useCapturedValues (initial/final value tracking)
 * - useProfilingAnalyzers (leak, FPS, CPU, GPU analysis)
 * - useProfilingRange (range CRUD + metadata writes)
 *
 * Handles:
 * - Status transitions (start, pause, resume, reset)
 * - Dispatching analysis reports to Redux
 * - Coordinating range lifecycle with status changes
 */
export const useProfilingSession = ({
  status,
  collectors,
  sampleBuffer,
  getAggregates,
  resetEventCollectors,
  resetTableData,
  resetBuffer,
}: UseProfilingSessionOptions): UseProfilingSessionResult => {
  const dispatch = useDispatch();
  const prevStatusRef = useRef<HarnessStatus>(status);

  // Store range data in refs so it survives the idle transition
  const rangeDataRef = useRef<{ key: string; startTime: number } | null>(null);

  const { captured, captureInitial, captureFinal, reset: resetCaptured } = useCapturedValues();
  const { analyze } = useProfilingAnalyzers();
  const {
    rangeKey,
    rangeStartTime,
    createRange,
    updateEndTime,
    clearStopValues,
    finalizeRange,
    addMetricLabel,
    removeTransientLabel,
    isMetricLatched,
  } = useProfilingRange({
    status,
    getMetrics: useCallback(() => {
      const agg = getAggregates();
      return {
        averages: { cpu: agg.avgCpu, fps: agg.avgFps, gpu: agg.avgGpu },
        peaks: {
          cpu: agg.maxCpu,
          fps: agg.minFps,
          gpu: agg.maxGpu,
          heap: agg.maxHeap,
        },
      };
    }, [getAggregates]),
  });

  const handleSample = useCallback(
    (sample: MetricSample, buffer: SampleBuffer) => {
      const cap = captured.current;

      if (cap.initialCPU == null && sample.cpuPercent != null) {
        cap.initialCPU = sample.cpuPercent;
        dispatch(
          Perf.setCpuReport({
            ...ZERO_CPU_REPORT,
            startPercent: math.roundTo(sample.cpuPercent),
          }),
        );
      }
      if (cap.initialGPU == null && sample.gpuPercent != null) {
        cap.initialGPU = sample.gpuPercent;
        dispatch(
          Perf.setGpuReport({
            ...ZERO_GPU_REPORT,
            startPercent: math.roundTo(sample.gpuPercent),
          }),
        );
      }

      const allSamples = buffer.getAllSamples();
      if (allSamples.length < 2) return;

      const results = analyze({
        samples: allSamples,
        currentSample: sample,
        captured: cap,
        aggregates: buffer.getAggregates(),
      });

      // Batch update reports to minimize redux action dispatching
      dispatch(
        Perf.setReports({
          leak: results.leak,
          fps: results.fps,
          cpu: results.cpu,
          gpu: results.gpu,
        }),
      );

      // Add real-time labels with latching behavior:
      // - Peak-triggered labels are latched (permanent)
      // - Avg-triggered labels are transient (can be removed if avg improves)
      // - Once latched, skip further calculations for that metric
      for (const metric of ["fps", "cpu", "gpu"] as const) {
        // Skip if already latched
        if (isMetricLatched(metric)) continue;

        const report =
          metric === "fps" ? results.fps : metric === "cpu" ? results.cpu : results.gpu;

        // First check peak - if triggered, latch and skip avg check
        if (report.peakSeverity !== "none") {
          addMetricLabel({ metric, severity: report.peakSeverity, latched: true });
          continue;
        }

        // Peak not triggered - check avg (transient, can be removed)
        if (report.avgSeverity !== "none") 
          addMetricLabel({ metric, severity: report.avgSeverity, latched: false });
         else 
          removeTransientLabel({ metric });
        
      }

      // Heap uses single severity (no peak/avg distinction)
      if (!isMetricLatched("heap") && results.leak.severity !== "none") 
        addMetricLabel({ metric: "heap", severity: results.leak.severity, latched: true });
      

    },
    [dispatch, captured, analyze, addMetricLabel, removeTransientLabel, isMetricLatched],
  );

  useEffect(() => {
    const prevStatus = prevStatusRef.current;
    prevStatusRef.current = status;
    const c = collectors.current;

    if (status === "running" && prevStatus === "idle") {
      resetEventCollectors();
      resetTableData();

      const startValues = getCollectorValues(c);
      captureInitial(startValues);

      // Batch initial reports
      dispatch(
        Perf.setReports({
          fps: {
            ...ZERO_FPS_REPORT,
            startFps: startValues.fps != null ? math.roundTo(startValues.fps) : null,
          },
          leak: {
            ...ZERO_LEAK_REPORT,
            heapStartMB:
              startValues.heap != null ? math.roundTo(startValues.heap) : null,
          },
          cpu: {
            ...ZERO_CPU_REPORT,
            startPercent:
              startValues.cpu != null ? math.roundTo(startValues.cpu) : null,
          },
          gpu: {
            ...ZERO_GPU_REPORT,
            startPercent:
              startValues.gpu != null ? math.roundTo(startValues.gpu) : null,
          },
        }),
      );

      createRange({ startValues });
    }

    // Capture range data into ref when it becomes available
    if (status === "running" && rangeKey != null && rangeStartTime != null)
      rangeDataRef.current = { key: rangeKey, startTime: rangeStartTime };
    

    if (status === "paused" && prevStatus === "running") {
      const samples = sampleBuffer.current?.getAllSamples() ?? [];
      const lastSample = samples.at(-1) ?? null;

      captureFinal({ sample: lastSample, fallback: getCollectorValues(c) });

      updateEndTime(TimeStamp.now());
    }

    if (status === "running" && prevStatus === "paused") {
      updateEndTime(TimeStamp.MAX);
      clearStopValues();
    }
    

    if (status === "idle" && prevStatus !== "idle") {
      const rangeData = rangeDataRef.current;
      if ((prevStatus === "running" || prevStatus === "paused") && rangeData != null) {
        const agg = getAggregates();
        const samples = sampleBuffer.current?.getAllSamples() ?? [];
        const lastSample = samples.at(-1) ?? null;
        const cap = captured.current;

        const fallback = getCollectorValues(c);
        const stopValues: LiveValues = {
          fps: lastSample?.frameRate ?? fallback.fps,
          cpu: lastSample?.cpuPercent ?? fallback.cpu,
          gpu: lastSample?.gpuPercent ?? fallback.gpu,
          heap: lastSample?.heapUsedMB ?? fallback.heap,
        };

        // Run final analysis to get severities
        const results = analyze({
          samples,
          currentSample: lastSample ?? {
            timestamp: 0,
            cpuPercent: null,
            gpuPercent: null,
            heapUsedMB: null,
            heapTotalMB: null,
            frameRate: null,
            longTaskCount: 0,
            longTaskDurationMs: 0,
            networkRequestCount: 0,
            consoleLogCount: 0,
          },
          captured: cap,
          aggregates: agg,
        });

        finalizeRange({
          rangeKey: rangeData.key,
          startTime: rangeData.startTime,
          severities: {
            fps: { peak: results.fps.peakSeverity, avg: results.fps.avgSeverity },
            cpu: { peak: results.cpu.peakSeverity, avg: results.cpu.avgSeverity },
            gpu: { peak: results.gpu.peakSeverity, avg: results.gpu.avgSeverity },
            heap: results.leak.severity,
          },
          stopValues,
        });
      }

      rangeDataRef.current = null;

      resetCaptured();
      resetEventCollectors();
      resetTableData();
      resetBuffer();
    }
  }, [
    status,
    dispatch,
    collectors,
    sampleBuffer,
    rangeKey,
    rangeStartTime,
    captureInitial,
    captureFinal,
    resetCaptured,
    createRange,
    updateEndTime,
    clearStopValues,
    finalizeRange,
    getAggregates,
    resetEventCollectors,
    resetTableData,
    resetBuffer,
    analyze,
    captured,
    // Note: rangeDataRef is intentionally NOT in deps - it's a ref that persists across renders
  ]);

  return { handleSample };
};
