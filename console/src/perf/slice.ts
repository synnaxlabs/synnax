// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type deep } from "@synnaxlabs/x";

import {
  type CpuReport,
  type DegradationReport,
  type GpuReport,
  type LeakReport,
  ZERO_CPU_REPORT,
  ZERO_DEGRADATION_REPORT,
  ZERO_GPU_REPORT,
  ZERO_LEAK_REPORT,
} from "@/perf/analyzer/types";
import { DEFAULT_METRICS_CONFIG, type MetricsConfig } from "@/perf/metrics/types";
import {
  DEFAULT_WORKFLOW_CONFIG,
  type WorkflowConfig,
  type WorkflowResult,
} from "@/perf/workflows/types";

export const SLICE_NAME = "perf";

export type HarnessStatus = "idle" | "running" | "paused" | "error";

/** Configuration for the performance harness. */
export interface HarnessConfig {
  durationMinutes: number;
  metricsConfig: MetricsConfig;
  workflowConfig: WorkflowConfig;
}

export const DEFAULT_HARNESS_CONFIG: HarnessConfig = {
  durationMinutes: 30,
  metricsConfig: DEFAULT_METRICS_CONFIG,
  workflowConfig: DEFAULT_WORKFLOW_CONFIG,
};

/** Redux slice state for performance harness. */
export interface SliceState {
  status: HarnessStatus;
  config: HarnessConfig;
  workflowResults: WorkflowResult[];
  error: string | null;
  startTime: number | null;
  endTime: number | null;
  rangeKey: string | null;
  rangeStartTime: number | null;
  leakReport: LeakReport;
  degradationReport: DegradationReport;
  cpuReport: CpuReport;
  gpuReport: GpuReport;
}

export const ZERO_SLICE_STATE: SliceState = {
  status: "idle",
  config: DEFAULT_HARNESS_CONFIG,
  workflowResults: [],
  error: null,
  startTime: null,
  endTime: null,
  rangeKey: null,
  rangeStartTime: null,
  leakReport: ZERO_LEAK_REPORT,
  degradationReport: ZERO_DEGRADATION_REPORT,
  cpuReport: ZERO_CPU_REPORT,
  gpuReport: ZERO_GPU_REPORT,
};

/** Store state shape for the perf slice. */
export interface StoreState {
  [SLICE_NAME]: SliceState;
}

/**
 * Fields to exclude from persistence.
 * Perf data is transient and shouldn't be saved to disk between app restarts.
 */
export const PERSIST_EXCLUDE = [
  "config",
  "workflowResults",
  "startTime",
  "endTime",
  "rangeKey",
  "rangeStartTime",
  "leakReport",
  "degradationReport",
  "cpuReport",
  "gpuReport",
  "status",
  "error",
].map((key) => `${SLICE_NAME}.${key}`) as Array<deep.Key<StoreState>>;

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    start: (state, { payload }: PayloadAction<Partial<HarnessConfig> | undefined>) => {
      state.status = "running";
      if (payload != null)
        state.config = {
          ...state.config,
          ...payload,
          metricsConfig: { ...state.config.metricsConfig, ...payload.metricsConfig },
          workflowConfig: { ...state.config.workflowConfig, ...payload.workflowConfig },
        };
      state.startTime = performance.now();
      state.endTime = null;
      state.rangeKey = null;
      state.rangeStartTime = null;
      state.workflowResults = [];
      state.error = null;
      state.leakReport = ZERO_LEAK_REPORT;
      state.degradationReport = ZERO_DEGRADATION_REPORT;
      state.cpuReport = ZERO_CPU_REPORT;
      state.gpuReport = ZERO_GPU_REPORT;
    },

    stop: (state) => {
      // Stop now behaves the same as pause - we no longer have a "completed" state
      if (state.status === "running") state.status = "paused";
    },

    pause: (state) => {
      if (state.status === "running") state.status = "paused";
    },

    resume: (state) => {
      if (state.status === "paused") state.status = "running";
    },

    addWorkflowResult: (state, { payload }: PayloadAction<WorkflowResult>) => {
      state.workflowResults.push(payload);
    },

    setLeakReport: (state, { payload }: PayloadAction<LeakReport>) => {
      state.leakReport = payload;
    },

    setDegradationReport: (state, { payload }: PayloadAction<DegradationReport>) => {
      state.degradationReport = payload;
    },

    setCpuReport: (state, { payload }: PayloadAction<CpuReport>) => {
      state.cpuReport = payload;
    },

    setGpuReport: (state, { payload }: PayloadAction<GpuReport>) => {
      state.gpuReport = payload;
    },

    setRangeKey: (state, { payload }: PayloadAction<string | null>) => {
      state.rangeKey = payload;
    },

    setRangeStartTime: (state, { payload }: PayloadAction<number | null>) => {
      state.rangeStartTime = payload;
    },

    setError: (state, { payload }: PayloadAction<string>) => {
      state.status = "error";
      state.error = payload;
      state.endTime = performance.now();
    },

    reset: () => ZERO_SLICE_STATE,

    setConfig: (state, { payload }: PayloadAction<Partial<HarnessConfig>>) => {
      state.config = {
        ...state.config,
        ...payload,
        metricsConfig: { ...state.config.metricsConfig, ...payload.metricsConfig },
        workflowConfig: { ...state.config.workflowConfig, ...payload.workflowConfig },
      };
    },
  },
});

export const {
  start,
  stop,
  pause,
  resume,
  addWorkflowResult,
  setLeakReport,
  setDegradationReport,
  setCpuReport,
  setGpuReport,
  setRangeKey,
  setRangeStartTime,
  setError,
  reset,
  setConfig,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
