// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CpuReport } from "@/perf/analyzer/types";
import { THRESHOLDS } from "@/perf/constants";
import { type MetricDef } from "@/perf/types";
import {
  formatDelta,
  formatFps,
  formatMB,
  formatPair,
  formatPercent,
  formatPercentChange,
} from "@/perf/utils/formatting";
import { getAvgPeakStatus, getThresholdStatus } from "@/perf/utils/status";

export type ResourceReport = Omit<CpuReport, "detected">;

export const createFpsMetrics = (
  liveValue: number | null,
  degradationPercent: number | null,
  hasData: boolean,
  avgFps: number | null,
  minFps: number | null,
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
        THRESHOLDS.fpsDegradation.warn,
        THRESHOLDS.fpsDegradation.error,
      ),
    tooltip: `FPS change during session (warn >${THRESHOLDS.fpsDegradation.warn}%, error >${THRESHOLDS.fpsDegradation.error}%)`,
  },
  {
    key: "fps-stats",
    type: "fps",
    category: "stats",
    getValue: () => formatPair(avgFps, minFps),
    getStatus: () =>
      getThresholdStatus(
        minFps,
        THRESHOLDS.fps.warn,
        THRESHOLDS.fps.error,
        THRESHOLDS.fps.inverted,
      ),
    tooltip: `Average and minimum FPS (warn <${THRESHOLDS.fps.warn}, error <${THRESHOLDS.fps.error})`,
  },
];

export const createMemoryMetrics = (
  liveHeap: number | null,
  growthPercent: number | null,
  hasData: boolean,
  avgHeap: number | null,
  peakHeap: number | null,
): MetricDef[] => [
  {
    key: "memory-live",
    type: "memory",
    category: "live",
    getValue: () => formatMB(liveHeap),
    tooltip: "Current heap usage",
  },
  {
    key: "memory-change",
    type: "memory",
    category: "change",
    getValue: () => formatPercentChange(hasData ? growthPercent : null),
    getStatus: () =>
      getThresholdStatus(
        growthPercent,
        THRESHOLDS.heapGrowth.warn,
        THRESHOLDS.heapGrowth.error,
      ),
    tooltip: `Heap change during session (warn >${THRESHOLDS.heapGrowth.warn}%, error >${THRESHOLDS.heapGrowth.error}%)`,
  },
  {
    key: "memory-stats",
    type: "memory",
    category: "stats",
    getValue: () => formatPair(avgHeap, peakHeap, " MB"),
    tooltip: "Average and max heap usage",
    label: "Avg / Max",
  },
];

export const createResourceMetrics = (
  type: "cpu" | "gpu",
  liveValue: number | null,
  report: ResourceReport,
  thresholds: { warn: number; error: number },
  changeThresholds: { warn: number; error: number },
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
      getValue: () => formatPair(report.avgPercent, report.peakPercent, "%"),
      getStatus: () =>
        getAvgPeakStatus(
          report.avgPercent,
          report.peakPercent,
          thresholds.warn,
          thresholds.error,
        ),
      tooltip: `Average and max ${label} (warn >${thresholds.warn}%, error >${thresholds.error}%)`,
      label: "Avg / Max",
    },
  ];
};
