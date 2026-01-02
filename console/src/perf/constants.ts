// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export const DISPLAY_LIMIT = 25;
export const RESIZE_THRESHOLD = 6;
export const LIVE_DISPLAY_INTERVAL_MS = 1000;
export const SAMPLE_INTERVAL_MS = 1000;
export const WARMUP_SAMPLES = 5;
export const BASELINE_BUFFER_SIZE = 60;
export const RECENT_BUFFER_SIZE = 60;
export const TEXT_ROW_COLOR = 7 as const;

export const LONG_TASK_THRESHOLD_MS = 50;
export const EVENT_CORRELATION_WINDOW_MS = 1000;
export const LONG_TASK_WINDOW_MS = 600_000;
export const MAX_STORED_ENDPOINTS = 100;
export const MAX_TRACKED_EVENTS = 50;
export const CONSOLE_LOG_WINDOW_MS = 600_000;
export const MAX_STORED_MESSAGES = 1000;
export const MAX_MESSAGE_LENGTH = 500;

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

export type MetricType = "fps" | "heap" | "cpu" | "gpu";
export type MetricCategory = "live" | "change" | "stats";

export const METRIC_NAMES: Record<MetricType, string> = {
  fps: "FPS",
  heap: "Heap",
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

export const METRIC_ORDER: MetricType[] = ["fps", "heap", "cpu", "gpu"];
export const CATEGORY_ORDER: MetricCategory[] = ["live", "change", "stats"];

export const HEAP_COMPARISON_WINDOW_SIZE = 30;
export const HEAP_SLOPE_THRESHOLD = 0.1;

export const THRESHOLDS = {
  fpsAvg: { warn: 25, error: 10, inverted: true },
  cpuAvg: { warn: 75, error: 95 },
  gpuAvg: { warn: 75, error: 95 },
  fpsChange: { warn: 50, error: 80, inverted: true },
  cpuChange: { warn: 80, error: 98 },
  gpuChange: { warn: 80, error: 98 },
  heapGrowth: { warn: 20, error: 40 },
  fps: { warn: 10, error: 1, inverted: true },
  cpu: { warn: 90, error: 99 },
  gpu: { warn: 90, error: 99 },
} as const;

export const LABEL_COLORS = {
  nominal: "#3774d0", // --pluto-primary-z
  warning: "#c29d0a", // --pluto-warning-m1
  error: "#f5242e", // --pluto-error-z
} as const;

export const NOMINAL_LABEL_NAME = "Nominal";

export interface LabelConfig {
  name: string;
  color: string;
}

export const getMetricLabelName = (
  metric: MetricType,
  severity: "warning" | "error",
): string => {
  const suffix = severity === "error" ? "Error" : "Warn";
  return `${METRIC_NAMES[metric]} ${suffix}`;
};

export const getProfilingLabelConfigs = (): LabelConfig[] => {
  const labels: LabelConfig[] = [
    { name: NOMINAL_LABEL_NAME, color: LABEL_COLORS.nominal },
  ];
  for (const metric of METRIC_ORDER) {
    labels.push({
      name: getMetricLabelName(metric, "warning"),
      color: LABEL_COLORS.warning,
    });
    labels.push({
      name: getMetricLabelName(metric, "error"),
      color: LABEL_COLORS.error,
    });
  }
  return labels;
};
