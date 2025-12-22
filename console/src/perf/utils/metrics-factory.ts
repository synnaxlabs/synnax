// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CpuReport, type Severity } from "@/perf/analyzer/types";
import { THRESHOLDS } from "@/perf/constants";
import { type MetricDef, type Status } from "@/perf/types";
import {
  formatDelta,
  formatFps,
  formatMB,
  formatPair,
  formatPercent,
  formatPercentChange,
} from "@/perf/utils/formatting";
import { getThresholdStatus } from "@/perf/utils/status";

export type ResourceReport = Omit<CpuReport, "detected">;

export interface MetricSeverities {
  peakSeverity: Severity;
  avgSeverity: Severity;
}

const severityToStatus = (severities: MetricSeverities): Status => {
  if (severities.peakSeverity === "error" || severities.avgSeverity === "error")
    return "error";
  if (severities.peakSeverity === "warning" || severities.avgSeverity === "warning")
    return "warning";
  return undefined;
};

export const createFpsMetrics = (
  liveValue: number | null,
  degradationPercent: number | null,
  hasData: boolean,
  avgFps: number | null,
  minFps: number | null,
  severities: MetricSeverities,
): MetricDef[] => [
  {
    key: "fps-live",
    type: "fps",
    category: "live",
    getValue: () => formatFps(liveValue),
    getStatus: () =>
      getThresholdStatus(
        liveValue,
        THRESHOLDS.fps.warn,
        THRESHOLDS.fps.error,
        THRESHOLDS.fps.inverted,
      ),
    tooltip: `Current FPS (warn <${THRESHOLDS.fps.warn}, error <${THRESHOLDS.fps.error})`,
  },
  {
    key: "fps-change",
    type: "fps",
    category: "change",
    getValue: () => formatPercentChange(hasData ? degradationPercent : null, true),
    getStatus: () =>
      getThresholdStatus(
        degradationPercent,
        THRESHOLDS.fpsChange.warn,
        THRESHOLDS.fpsChange.error,
      ),
    tooltip: `FPS change during session (warn >${THRESHOLDS.fpsChange.warn}%, error >${THRESHOLDS.fpsChange.error}%)`,
  },
  {
    key: "fps-stats",
    type: "fps",
    category: "stats",
    getValue: () => formatPair(avgFps, minFps),
    getStatus: () => severityToStatus(severities),
    tooltip: `Average (warn <${THRESHOLDS.fpsAvg.warn}, error <${THRESHOLDS.fpsAvg.error}) and minimum FPS (warn <${THRESHOLDS.fps.warn}, error <${THRESHOLDS.fps.error})`,
  },
];

/** Convert single severity to UI Status */
const singleSeverityToStatus = (severity: Severity): Status => {
  if (severity === "error") return "error";
  if (severity === "warning") return "warning";
  return undefined;
};

export const createMemoryMetrics = (
  liveHeap: number | null,
  growthPercent: number | null,
  hasData: boolean,
  minHeap: number | null,
  maxHeap: number | null,
  severity: Severity,
): MetricDef[] => [
  {
    key: "heap-live",
    type: "heap",
    category: "live",
    getValue: () => formatMB(liveHeap),
    tooltip: "Current heap usage",
  },
  {
    key: "heap-change",
    type: "heap",
    category: "change",
    getValue: () => formatPercentChange(hasData ? growthPercent : null),
    getStatus: () => singleSeverityToStatus(severity),
    tooltip: `Heap change during session (warn >${THRESHOLDS.heapGrowth.warn}%, error >${THRESHOLDS.heapGrowth.error}%)`,
  },
  {
    key: "heap-stats",
    type: "heap",
    category: "stats",
    getValue: () => formatPair(minHeap, maxHeap, " MB"),
    tooltip: "Min and max heap usage",
    label: "Min / Max",
  },
];

export const createResourceMetrics = (
  type: "cpu" | "gpu",
  liveValue: number | null,
  report: ResourceReport,
  thresholds: { warn: number; error: number },
  changeThresholds: { warn: number; error: number },
  severities: MetricSeverities,
): MetricDef[] => {
  const label = type.toUpperCase();
  return [
    {
      key: `${type}-live`,
      type,
      category: "live",
      getValue: () => formatPercent(liveValue),
      getStatus: () => getThresholdStatus(liveValue, thresholds.warn, thresholds.error),
      tooltip: `Current ${label} (warn >${thresholds.warn}%, error >${thresholds.error}%)`,
    },
    {
      key: `${type}-change`,
      type,
      category: "change",
      getValue: () => formatDelta(report.startPercent, report.endPercent, "%"),
      getStatus: () => {
        if (report.startPercent == null || report.endPercent == null) return undefined;
        const delta = report.endPercent - report.startPercent;
        if (delta < 0) return undefined;
        return getThresholdStatus(delta, changeThresholds.warn, changeThresholds.error);
      },
      tooltip: `${label} change during session (warn >${changeThresholds.warn}%, error >${changeThresholds.error}%)`,
    },
    {
      key: `${type}-stats`,
      type,
      category: "stats",
      getValue: () => formatPair(report.avgPercent, report.maxPercent, "%"),
      getStatus: () => severityToStatus(severities),
      tooltip: `Average (warn >${type === "cpu" ? THRESHOLDS.cpuAvg.warn : THRESHOLDS.gpuAvg.warn}%, error >${type === "cpu" ? THRESHOLDS.cpuAvg.error : THRESHOLDS.gpuAvg.error}%) and max ${label} (warn >${thresholds.warn}%, error >${thresholds.error}%)`,
      label: "Avg / Max",
    },
  ];
};
