// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useSelector } from "react-redux";

import { type DegradationReport, generateReport, type LeakReport, type PerfReport } from "@/perf/analyzer/types";
import { type HeapSnapshot,type MetricSample } from "@/perf/metrics/types";
import { type HarnessStatus,SLICE_NAME, type SliceState } from "@/perf/slice";
import { type WorkflowResult } from "@/perf/workflows/types";
import { type RootState } from "@/store";

/** Select the entire perf slice state. */
export const selectSlice = (state: RootState): SliceState => state[SLICE_NAME];

/** Select the harness status. */
export const selectStatus = (state: RootState): HarnessStatus =>
  selectSlice(state).status;

/** Select whether the harness is running. */
export const selectIsRunning = (state: RootState): boolean =>
  selectSlice(state).status === "running";

/** Select collected metric samples. */
export const selectSamples = (state: RootState): MetricSample[] =>
  selectSlice(state).samples;

/** Select collected heap snapshots. */
export const selectHeapSnapshots = (state: RootState): HeapSnapshot[] =>
  selectSlice(state).heapSnapshots;

/** Select workflow results. */
export const selectWorkflowResults = (state: RootState): WorkflowResult[] =>
  selectSlice(state).workflowResults;

/** Select the leak report. */
export const selectLeakReport = (state: RootState): LeakReport =>
  selectSlice(state).leakReport;

/** Select the degradation report. */
export const selectDegradationReport = (state: RootState): DegradationReport =>
  selectSlice(state).degradationReport;

/** Select the error message. */
export const selectError = (state: RootState): string | null =>
  selectSlice(state).error;

/** Select the latest metric sample. */
export const selectLatestSample = (state: RootState): MetricSample | null => {
  const samples = selectSamples(state);
  return samples.length > 0 ? samples[samples.length - 1] : null;
};

/** Select elapsed time in seconds. */
export const selectElapsedSeconds = (state: RootState): number => {
  const slice = selectSlice(state);
  if (slice.startTime == null) return 0;
  const endTime = slice.endTime ?? performance.now();
  return (endTime - slice.startTime) / 1000;
};

/** Generate a full performance report from current state. */
export const selectReport = (state: RootState): PerfReport | null => {
  const slice = selectSlice(state);
  if (slice.startTime == null || slice.samples.length === 0) return null;

  return generateReport(
    slice.samples,
    slice.heapSnapshots,
    slice.workflowResults,
    slice.startTime,
    slice.endTime ?? performance.now(),
    slice.leakReport,
    slice.degradationReport,
  );
};

// React hooks for selectors
export const useSelectSlice = (): SliceState => useSelector(selectSlice);
export const useSelectStatus = (): HarnessStatus => useSelector(selectStatus);
export const useSelectIsRunning = (): boolean => useSelector(selectIsRunning);
export const useSelectSamples = (): MetricSample[] => useSelector(selectSamples);
export const useSelectHeapSnapshots = (): HeapSnapshot[] => useSelector(selectHeapSnapshots);
export const useSelectWorkflowResults = (): WorkflowResult[] => useSelector(selectWorkflowResults);
export const useSelectLeakReport = (): LeakReport => useSelector(selectLeakReport);
export const useSelectDegradationReport = (): DegradationReport => useSelector(selectDegradationReport);
export const useSelectError = (): string | null => useSelector(selectError);
export const useSelectLatestSample = (): MetricSample | null => useSelector(selectLatestSample);
export const useSelectElapsedSeconds = (): number => useSelector(selectElapsedSeconds);
export const useSelectReport = (): PerfReport | null => useSelector(selectReport);
