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
import { useProfilingAnalyzers } from "@/perf/hooks/useProfilingAnalyzers";
import { useProfilingRange } from "@/perf/hooks/useProfilingRange";
import { type Aggregates, type SampleBuffer } from "@/perf/metrics/buffer";
import { type MetricSample } from "@/perf/metrics/types";
import * as Perf from "@/perf/slice";
import { type HarnessStatus } from "@/perf/slice";

/** Collector state interface matching useCollectors. */
interface CollectorsState {
  cpu: { getCpuPercent: () => number | null } | null;
  gpu: { getGpuPercent: () => number | null } | null;
  frameRate: { getCurrentFPS: () => number | null } | null;
  heap: { getHeapUsedMB: () => number | null } | null;
  longTask: unknown;
  network: unknown;
  console: unknown;
}

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

  // Compose child hooks
  const { captured, captureInitial, captureFinal, reset: resetCaptured } = useCapturedValues();
  const { analyze } = useProfilingAnalyzers();
  const { rangeKey, createRange, updateEndTime } = useProfilingRange({
    status,
    getMetadata: useCallback(() => {
      const agg = getAggregates();
      return {
        avgFps: agg.avgFps,
        avgCpu: agg.avgCpu,
        avgGpu: agg.avgGpu,
        avgHeap: agg.maxHeap,
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

      dispatch(Perf.setLeakReport(results.leak));
      dispatch(Perf.setFpsReport(results.fps));
      dispatch(Perf.setCpuReport(results.cpu));
      dispatch(Perf.setGpuReport(results.gpu));
    },
    [dispatch, captured, analyze],
  );

  useEffect(() => {
    const prevStatus = prevStatusRef.current;
    prevStatusRef.current = status;
    const c = collectors.current;

    if (status === "running" && prevStatus === "idle") {
      resetEventCollectors();
      resetTableData();

      const initialFPS = c?.frameRate?.getCurrentFPS() ?? null;
      const initialCPU = c?.cpu?.getCpuPercent() ?? null;
      const initialGPU = c?.gpu?.getGpuPercent() ?? null;
      const initialHeap = c?.heap?.getHeapUsedMB() ?? null;

      captureInitial({ fps: initialFPS, cpu: initialCPU, gpu: initialGPU, heap: initialHeap });

      dispatch(
        Perf.setFpsReport({
          ...ZERO_FPS_REPORT,
          startFps: initialFPS ?? 0,
        }),
      );
      dispatch(
        Perf.setLeakReport({
          ...ZERO_LEAK_REPORT,
          heapStartMB: math.roundTo(initialHeap ?? 0),
        }),
      );
      dispatch(
        Perf.setCpuReport({
          ...ZERO_CPU_REPORT,
          startPercent: initialCPU != null ? math.roundTo(initialCPU) : null,
        }),
      );
      dispatch(
        Perf.setGpuReport({
          ...ZERO_GPU_REPORT,
          startPercent: initialGPU != null ? math.roundTo(initialGPU) : null,
        }),
      );

      createRange();
    }

    if (status === "running" && prevStatus === "running" && rangeKey == null) {
      console.warn("[useProfilingSession] Range creation failed, retrying...");
      createRange();
    }
    

    if (status === "paused" && prevStatus === "running") {
      const samples = sampleBuffer.current?.getAllSamples() ?? [];
      const lastSample = samples.at(-1) ?? null;

      captureFinal({
        sample: lastSample,
        fallback: {
          fps: c?.frameRate?.getCurrentFPS() ?? null,
          cpu: c?.cpu?.getCpuPercent() ?? null,
          gpu: c?.gpu?.getGpuPercent() ?? null,
          heap: c?.heap?.getHeapUsedMB() ?? null,
        },
      });

      updateEndTime(TimeStamp.now());
    }

    if (status === "running" && prevStatus === "paused") 
      updateEndTime(TimeStamp.MAX);
    

    if (status === "idle" && prevStatus !== "idle") {
      if (prevStatus === "running" || prevStatus === "paused") 
        updateEndTime(TimeStamp.now());
      

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
    captureInitial,
    captureFinal,
    resetCaptured,
    createRange,
    updateEndTime,
    resetEventCollectors,
    resetTableData,
    resetBuffer,
  ]);

  return { handleSample };
};
