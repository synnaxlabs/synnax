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
  type DegradationReport,
  type LeakReport,
  ZERO_DEGRADATION_REPORT,
  ZERO_LEAK_REPORT,
} from "@/perf/analyzer/types";
import {
  DEFAULT_METRICS_CONFIG,
  type HeapSnapshot,
  type MetricSample,
  type MetricsConfig,
} from "@/perf/metrics/types";
import {
  DEFAULT_WORKFLOW_CONFIG,
  type WorkflowConfig,
  type WorkflowResult,
} from "@/perf/workflows/types";

export const SLICE_NAME = "perf";

/** Status of the performance harness. */
export type HarnessStatus = "idle" | "running" | "paused" | "completed" | "error";

/** Configuration for the performance harness. */
export interface HarnessConfig {
  /** Total duration to run in minutes (-1 for unlimited). */
  durationMinutes: number;
  /** Metrics collection configuration. */
  metricsConfig: MetricsConfig;
  /** Workflow execution configuration. */
  workflowConfig: WorkflowConfig;
}

/** Default harness configuration. */
export const DEFAULT_HARNESS_CONFIG: HarnessConfig = {
  durationMinutes: 30,
  metricsConfig: DEFAULT_METRICS_CONFIG,
  workflowConfig: DEFAULT_WORKFLOW_CONFIG,
};

/** Redux slice state for performance harness. */
export interface SliceState {
  /** Current status of the harness. */
  status: HarnessStatus;
  /** Harness configuration. */
  config: HarnessConfig;
  /** Collected metric samples. */
  samples: MetricSample[];
  /** Collected heap snapshots. */
  heapSnapshots: HeapSnapshot[];
  /** Workflow execution results. */
  workflowResults: WorkflowResult[];
  /** Error message if status is "error". */
  error: string | null;
  /** Start timestamp (performance.now()). */
  startTime: number | null;
  /** End timestamp (performance.now()). */
  endTime: number | null;
  /** Latest leak report. */
  leakReport: LeakReport;
  /** Latest degradation report. */
  degradationReport: DegradationReport;
}

/** Initial/zero state for the slice. */
export const ZERO_SLICE_STATE: SliceState = {
  status: "idle",
  config: DEFAULT_HARNESS_CONFIG,
  samples: [],
  heapSnapshots: [],
  workflowResults: [],
  error: null,
  startTime: null,
  endTime: null,
  leakReport: ZERO_LEAK_REPORT,
  degradationReport: ZERO_DEGRADATION_REPORT,
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
  "samples",
  "heapSnapshots",
  "workflowResults",
  "startTime",
  "endTime",
  "leakReport",
  "degradationReport",
  "status",
  "error",
].map((key) => `${SLICE_NAME}.${key}`) as Array<deep.Key<StoreState>>;

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    /** Start the performance harness with optional config overrides. */
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
      state.samples = [];
      state.heapSnapshots = [];
      state.workflowResults = [];
      state.error = null;
      state.leakReport = ZERO_LEAK_REPORT;
      state.degradationReport = ZERO_DEGRADATION_REPORT;
    },

    /** Stop the performance harness. */
    stop: (state) => {
      state.status = "completed";
      state.endTime = performance.now();
    },

    /** Pause the performance harness. */
    pause: (state) => {
      if (state.status === "running") 
        state.status = "paused";
      
    },

    /** Resume a paused harness. */
    resume: (state) => {
      if (state.status === "paused") 
        state.status = "running";
      
    },

    /** Add a new metric sample. */
    addSample: (state, { payload }: PayloadAction<MetricSample>) => {
      state.samples.push(payload);
    },

    /** Add a new heap snapshot. */
    addHeapSnapshot: (state, { payload }: PayloadAction<HeapSnapshot>) => {
      state.heapSnapshots.push(payload);
    },

    /** Add a workflow result. */
    addWorkflowResult: (state, { payload }: PayloadAction<WorkflowResult>) => {
      state.workflowResults.push(payload);
    },

    /** Update the leak report. */
    setLeakReport: (state, { payload }: PayloadAction<LeakReport>) => {
      state.leakReport = payload;
    },

    /** Update the degradation report. */
    setDegradationReport: (state, { payload }: PayloadAction<DegradationReport>) => {
      state.degradationReport = payload;
    },

    /** Set an error state. */
    setError: (state, { payload }: PayloadAction<string>) => {
      state.status = "error";
      state.error = payload;
      state.endTime = performance.now();
    },

    /** Reset the harness to initial state. */
    reset: () => ZERO_SLICE_STATE,

    /** Update configuration without starting. */
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
  addSample,
  addHeapSnapshot,
  addWorkflowResult,
  setLeakReport,
  setDegradationReport,
  setError,
  reset,
  setConfig,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
