// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type MacroType } from "@/perf/macros/types";

export type MacroExecutionStatus = "idle" | "running" | "cancelled";

export interface MacroExecutionProgress {
  currentIteration: number;
  totalIterations: number;
  currentMacro: MacroType | null;
  currentMacroIndex: number;
  totalMacros: number;
}

export interface MacroExecutionState {
  status: MacroExecutionStatus;
  progress: MacroExecutionProgress;
}

export const ZERO_EXECUTION_STATE: MacroExecutionState = {
  status: "idle",
  progress: {
    currentIteration: 0,
    totalIterations: 0,
    currentMacro: null,
    currentMacroIndex: 0,
    totalMacros: 0,
  },
};

export interface MacroRunConfig {
  macros: MacroType[];
  iterations: number;
}

export const DEFAULT_RUN_CONFIG: MacroRunConfig = {
  macros: [],
  iterations: 1,
};
