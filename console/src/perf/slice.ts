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
  type FpsReport,
  type GpuReport,
  type LeakReport,
  ZERO_CPU_REPORT,
  ZERO_FPS_REPORT,
  ZERO_GPU_REPORT,
  ZERO_LEAK_REPORT,
} from "@/perf/analyzer/types";
import { type MacroResult } from "@/perf/macros/types";
import * as latest from "@/perf/types";

export const SLICE_NAME = "perf";

// Re-export types from versioned types file (following lineplot pattern)
export type HarnessStatus = latest.HarnessStatus;
export type HarnessConfig = latest.HarnessConfig;
export type SliceState = Omit<latest.SliceState, "version">;
export const DEFAULT_HARNESS_CONFIG = latest.DEFAULT_HARNESS_CONFIG;
export const ZERO_SLICE_STATE: SliceState = {
  status: latest.ZERO_SLICE_STATE.status,
  config: latest.ZERO_SLICE_STATE.config,
  macroResults: latest.ZERO_SLICE_STATE.macroResults,
  error: latest.ZERO_SLICE_STATE.error,
  startTime: latest.ZERO_SLICE_STATE.startTime,
  endTime: latest.ZERO_SLICE_STATE.endTime,
  rangeKey: latest.ZERO_SLICE_STATE.rangeKey,
  rangeStartTime: latest.ZERO_SLICE_STATE.rangeStartTime,
  leakReport: latest.ZERO_SLICE_STATE.leakReport,
  fpsReport: latest.ZERO_SLICE_STATE.fpsReport,
  cpuReport: latest.ZERO_SLICE_STATE.cpuReport,
  gpuReport: latest.ZERO_SLICE_STATE.gpuReport,
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
  "macroResults",
  "startTime",
  "endTime",
  "rangeKey",
  "rangeStartTime",
  "leakReport",
  "fpsReport",
  "cpuReport",
  "gpuReport",
  "status",
  "error",
].map((key) => `${SLICE_NAME}.${key}`) as Array<deep.Key<StoreState>>;

export type PartialHarnessConfig = Partial<
  Omit<HarnessConfig, "metricsConfig" | "macroConfig">
> & {
  metricsConfig?: Partial<latest.MetricsConfig>;
  macroConfig?: Partial<latest.MacroConfig>;
};

export type StartPayload = PartialHarnessConfig | undefined;
export type AddMacroResultPayload = MacroResult;
export type SetLeakReportPayload = LeakReport;
export type SetFpsReportPayload = FpsReport;
export type SetCpuReportPayload = CpuReport;
export type SetGpuReportPayload = GpuReport;
export type SetRangeKeyPayload = string | null;
export type SetRangeStartTimePayload = number | null;
export type SetErrorPayload = string;
export type SetConfigPayload = PartialHarnessConfig;
export interface SetReportsPayload {
  leak: LeakReport;
  fps: FpsReport;
  cpu: CpuReport;
  gpu: GpuReport;
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    start: (state, { payload }: PayloadAction<StartPayload>) => {
      state.status = "running";
      if (payload != null)
        state.config = {
          ...state.config,
          ...payload,
          metricsConfig: { ...state.config.metricsConfig, ...payload.metricsConfig },
          macroConfig: { ...state.config.macroConfig, ...payload.macroConfig },
        };
      state.startTime = performance.now();
      state.endTime = null;
      state.rangeKey = null;
      state.rangeStartTime = null;
      state.macroResults = [];
      state.error = null;
      state.leakReport = ZERO_LEAK_REPORT;
      state.fpsReport = ZERO_FPS_REPORT;
      state.cpuReport = ZERO_CPU_REPORT;
      state.gpuReport = ZERO_GPU_REPORT;
    },

    stop: (state) => {
      if (state.status === "running") {
        state.status = "paused";
        state.endTime = performance.now();
      }
    },

    pause: (state) => {
      if (state.status === "running") {
        state.status = "paused";
        state.endTime = performance.now();
      }
    },

    resume: (state) => {
      if (state.status === "paused") {
        state.endTime = null;
        state.status = "running";
      }
    },

    addMacroResult: (state, { payload }: PayloadAction<AddMacroResultPayload>) => {
      state.macroResults.push(payload);
    },

    setLeakReport: (state, { payload }: PayloadAction<SetLeakReportPayload>) => {
      state.leakReport = payload;
    },

    setFpsReport: (state, { payload }: PayloadAction<SetFpsReportPayload>) => {
      state.fpsReport = payload;
    },

    setCpuReport: (state, { payload }: PayloadAction<SetCpuReportPayload>) => {
      state.cpuReport = payload;
    },

    setGpuReport: (state, { payload }: PayloadAction<SetGpuReportPayload>) => {
      state.gpuReport = payload;
    },

    /** Batch update all reports in a single action to reduce re-renders. */
    setReports: (state, { payload }: PayloadAction<SetReportsPayload>) => {
      state.leakReport = payload.leak;
      state.fpsReport = payload.fps;
      state.cpuReport = payload.cpu;
      state.gpuReport = payload.gpu;
    },

    setRangeKey: (state, { payload }: PayloadAction<SetRangeKeyPayload>) => {
      state.rangeKey = payload;
    },

    setRangeStartTime: (
      state,
      { payload }: PayloadAction<SetRangeStartTimePayload>,
    ) => {
      state.rangeStartTime = payload;
    },

    setError: (state, { payload }: PayloadAction<SetErrorPayload>) => {
      state.status = "error";
      state.error = payload;
      state.endTime = performance.now();
    },

    reset: () => ZERO_SLICE_STATE,

    setConfig: (state, { payload }: PayloadAction<SetConfigPayload>) => {
      state.config = {
        ...state.config,
        ...payload,
        metricsConfig: { ...state.config.metricsConfig, ...payload.metricsConfig },
        macroConfig: { ...state.config.macroConfig, ...payload.macroConfig },
      };
    },
  },
});

export const {
  start,
  stop,
  pause,
  resume,
  addMacroResult,
  setLeakReport,
  setFpsReport,
  setCpuReport,
  setGpuReport,
  setReports,
  setRangeKey,
  setRangeStartTime,
  setError,
  reset,
  setConfig,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export const migrateSlice = latest.migrateSlice;
