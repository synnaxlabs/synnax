// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

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
import { DEFAULT_METRICS_CONFIG, type MetricsConfig } from "@/perf/metrics/types";
import {
  DEFAULT_WORKFLOW_CONFIG,
  type WorkflowConfig,
  type WorkflowResult,
} from "@/perf/workflows/types";

export const VERSION = "0.0.0";

export const harnessStatusZ = z.enum(["idle", "running", "paused", "error"]);
export type HarnessStatus = z.infer<typeof harnessStatusZ>;

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

export interface SliceState {
  version: typeof VERSION;
  status: HarnessStatus;
  config: HarnessConfig;
  workflowResults: WorkflowResult[];
  error: string | null;
  startTime: number | null;
  endTime: number | null;
  rangeKey: string | null;
  rangeStartTime: number | null;
  leakReport: LeakReport;
  fpsReport: FpsReport;
  cpuReport: CpuReport;
  gpuReport: GpuReport;
}

export const ZERO_SLICE_STATE: SliceState = {
  version: VERSION,
  status: "idle",
  config: DEFAULT_HARNESS_CONFIG,
  workflowResults: [],
  error: null,
  startTime: null,
  endTime: null,
  rangeKey: null,
  rangeStartTime: null,
  leakReport: ZERO_LEAK_REPORT,
  fpsReport: ZERO_FPS_REPORT,
  cpuReport: ZERO_CPU_REPORT,
  gpuReport: ZERO_GPU_REPORT,
};

export type { CpuReport, FpsReport, GpuReport, LeakReport, MetricsConfig, WorkflowConfig };
