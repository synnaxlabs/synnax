// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** Performance profiling dashboard constants */

export const DISPLAY_LIMIT = 25;
export const LIVE_DISPLAY_INTERVAL_MS = 1000;
export const SAMPLE_INTERVAL_MS = 1000;
export const WARMUP_SAMPLES = 5;
export const LONG_TASK_THRESHOLD_MS = 50;
export const EVENT_CORRELATION_WINDOW_MS = 1000;
export const LONG_TASK_WINDOW_MS = 600_000;
export const MAX_STORED_ENDPOINTS = 100;
export const MAX_TRACKED_EVENTS = 50;

export const THRESHOLDS = {
  fps: { warn: 50, error: 28, inverted: true },
  fpsDegradation: { warn: 10, error: 15 },
  cpu: { warn: 25, error: 50 },
  cpuChange: { warn: 20, error: 40 },
  gpu: { warn: 25, error: 50 },
  gpuChange: { warn: 20, error: 40 },
  heapGrowth: { warn: 5, error: 10 },
  longTasks: { warn: 5, error: 10 },
  networkRequests: { warn: 5, error: 10 },
} as const;

export const STATUS_COLORS: Record<string, string> = {
  error: "var(--pluto-error-z)",
  warning: "var(--pluto-warning-z)",
  success: "var(--pluto-success-z)",
};

export const TEXT_ROW_COLOR = 7 as const;
export const TEXT_HEADER_COLOR = 9 as const;


export type MetricType = "fps" | "memory" | "cpu" | "gpu";
export type MetricCategory = "live" | "change" | "stats";

export const TYPE_LABELS: Record<MetricType, string> = {
  fps: "FPS",
  memory: "Memory",
  cpu: "CPU",
  gpu: "GPU",
};

export const CATEGORY_LABELS: Record<MetricCategory, string> = {
  live: "Live",
  change: "Change",
  stats: "Stats",
};

export const TYPE_MODE_LABELS: Record<MetricCategory, string> = {
  live: "Live",
  change: "Change",
  stats: "Avg / Min",
};

export const TYPE_ORDER: MetricType[] = ["fps", "memory", "cpu", "gpu"];
export const CATEGORY_ORDER: MetricCategory[] = ["live", "change", "stats"];

export const TRACKED_EVENT_TYPES = [
  "click",
  "keydown",
  "input",
  "submit",
  "dragstart",
  "dragend",
  "focus",
  "blur",
  "change",
  "paste",
  "scroll",
  "resize",
  "wheel",
  "touchstart",
  "touchend",
  "popstate",
  "contextmenu",
] as const;
