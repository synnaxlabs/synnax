// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useRef } from "react";

import { type MetricSample } from "@/perf/metrics/types";

/** Values captured at the start and end of a profiling session for delta calculations. */
export interface CapturedValues {
  initialFPS: number | null;
  finalFPS: number | null;
  initialCPU: number | null;
  finalCPU: number | null;
  initialGPU: number | null;
  finalGPU: number | null;
  initialHeap: number | null;
  finalHeap: number | null;
}

export const ZERO_CAPTURED_VALUES: CapturedValues = {
  initialFPS: null,
  finalFPS: null,
  initialCPU: null,
  finalCPU: null,
  initialGPU: null,
  finalGPU: null,
  initialHeap: null,
  finalHeap: null,
};

/** Input for capturing initial values from collectors. */
export interface CaptureInitialInput {
  fps: number | null;
  cpu: number | null;
  gpu: number | null;
  heap: number | null;
}

/** Input for capturing final values - prefers sample data, falls back to collector data. */
export interface CaptureFinalInput {
  sample: MetricSample | null;
  fallback: {
    fps: number | null;
    cpu: number | null;
    gpu: number | null;
    heap: number | null;
  };
}

export interface UseCapturedValuesResult {
  /** Ref to captured values - use ref for synchronous access in callbacks. */
  captured: React.RefObject<CapturedValues>;
  /** Capture initial values when profiling starts. */
  captureInitial: (input: CaptureInitialInput) => void;
  /** Capture final values when profiling pauses/stops. */
  captureFinal: (input: CaptureFinalInput) => void;
  /** Reset all captured values. */
  reset: () => void;
}

/**
 * Hook to track initial and final metric values for delta calculations.
 *
 * Used to compute changes like "FPS dropped 10%" or "CPU increased 5%"
 * between the start and end of a profiling session.
 */
export const useCapturedValues = (): UseCapturedValuesResult => {
  const capturedRef = useRef<CapturedValues>({ ...ZERO_CAPTURED_VALUES });

  const captureInitial = useCallback((input: CaptureInitialInput) => {
    const c = capturedRef.current;
    c.initialFPS = input.fps;
    c.finalFPS = null;
    c.initialCPU = input.cpu;
    c.finalCPU = null;
    c.initialGPU = input.gpu;
    c.finalGPU = null;
    c.initialHeap = input.heap;
    c.finalHeap = null;
  }, []);

  const captureFinal = useCallback((input: CaptureFinalInput) => {
    const c = capturedRef.current;
    const { sample, fallback } = input;

    if (sample != null) {
      c.finalFPS = sample.frameRate;
      c.finalCPU = sample.cpuPercent;
      c.finalGPU = sample.gpuPercent;
      c.finalHeap = sample.heapUsedMB;
    } else {
      c.finalFPS = fallback.fps;
      c.finalCPU = fallback.cpu;
      c.finalGPU = fallback.gpu;
      c.finalHeap = fallback.heap;
    }
  }, []);

  const reset = useCallback(() => {
    capturedRef.current = { ...ZERO_CAPTURED_VALUES };
  }, []);

  return {
    captured: capturedRef,
    captureInitial,
    captureFinal,
    reset,
  };
};
