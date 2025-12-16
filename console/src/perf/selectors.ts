// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useSelector } from "react-redux";

import {
  type CpuReport,
  type DegradationReport,
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

export const selectDegradationReport = (state: RootState): DegradationReport =>
  selectSlice(state).degradationReport;

export const selectCpuReport = (state: RootState): CpuReport =>
  selectSlice(state).cpuReport;

export const selectError = (state: RootState): string | null =>
  selectSlice(state).error;

export const selectElapsedSeconds = (state: RootState): number => {
  const slice = selectSlice(state);
  if (slice.startTime == null) return 0;
  const endTime = slice.endTime ?? performance.now();
  return (endTime - slice.startTime) / 1000;
};

export const useSelectSlice = (): SliceState => useSelector(selectSlice);
export const useSelectStatus = (): HarnessStatus => useSelector(selectStatus);
export const useSelectIsRunning = (): boolean => useSelector(selectIsRunning);
export const useSelectWorkflowResults = (): WorkflowResult[] =>
  useSelector(selectWorkflowResults);
export const useSelectLeakReport = (): LeakReport => useSelector(selectLeakReport);
export const useSelectDegradationReport = (): DegradationReport =>
  useSelector(selectDegradationReport);
export const useSelectCpuReport = (): CpuReport => useSelector(selectCpuReport);
export const useSelectError = (): string | null => useSelector(selectError);
export const useSelectElapsedSeconds = (): number => useSelector(selectElapsedSeconds);
