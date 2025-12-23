// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useMemoSelect } from "@/hooks";
import {
  type CpuReport,
  type FpsReport,
  type GpuReport,
  type LeakReport,
} from "@/perf/analyzer/types";
import { type HarnessStatus, SLICE_NAME, type SliceState } from "@/perf/slice";
import { type WorkflowResult } from "@/perf/workflows/types";
import { type RootState } from "@/store";

export const selectSlice = (state: RootState): SliceState => state[SLICE_NAME];

export const selectStatus = (state: RootState): HarnessStatus =>
  selectSlice(state).status;

export const selectIsRunning = (state: RootState): boolean =>
  selectSlice(state).status === "running";

export const selectWorkflowResults = (state: RootState): WorkflowResult[] =>
  selectSlice(state).workflowResults;

export const selectLeakReport = (state: RootState): LeakReport =>
  selectSlice(state).leakReport;

export const selectFpsReport = (state: RootState): FpsReport =>
  selectSlice(state).fpsReport;

export const selectCpuReport = (state: RootState): CpuReport =>
  selectSlice(state).cpuReport;

export const selectGpuReport = (state: RootState): GpuReport =>
  selectSlice(state).gpuReport;

export const selectError = (state: RootState): string | null =>
  selectSlice(state).error;

export const selectElapsedSeconds = (state: RootState): number => {
  const slice = selectSlice(state);
  if (slice.startTime == null) return 0;
  const endTime = slice.endTime ?? performance.now();
  return (endTime - slice.startTime) / 1000;
};

export const selectRangeKey = (state: RootState): string | null =>
  selectSlice(state).rangeKey;

export const selectRangeStartTime = (state: RootState): number | null =>
  selectSlice(state).rangeStartTime;

export const useSelectSlice = (): SliceState => useMemoSelect(selectSlice, []);
export const useSelectStatus = (): HarnessStatus => useMemoSelect(selectStatus, []);
export const useSelectIsRunning = (): boolean => useMemoSelect(selectIsRunning, []);
export const useSelectWorkflowResults = (): WorkflowResult[] =>
  useMemoSelect(selectWorkflowResults, []);
export const useSelectLeakReport = (): LeakReport => useMemoSelect(selectLeakReport, []);
export const useSelectFpsReport = (): FpsReport => useMemoSelect(selectFpsReport, []);
export const useSelectCpuReport = (): CpuReport => useMemoSelect(selectCpuReport, []);
export const useSelectGpuReport = (): GpuReport => useMemoSelect(selectGpuReport, []);
export const useSelectError = (): string | null => useMemoSelect(selectError, []);
export const useSelectElapsedSeconds = (): number =>
  useMemoSelect(selectElapsedSeconds, []);
export const useSelectRangeKey = (): string | null => useMemoSelect(selectRangeKey, []);
export const useSelectRangeStartTime = (): number | null =>
  useMemoSelect(selectRangeStartTime, []);
