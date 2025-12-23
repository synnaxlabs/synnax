// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type WorkflowType } from "@/perf/workflows/types";

export type WorkflowExecutionStatus = "idle" | "running" | "cancelled";

export interface WorkflowExecutionProgress {
  currentIteration: number;
  totalIterations: number;
  currentWorkflow: WorkflowType | null;
  currentWorkflowIndex: number;
  totalWorkflows: number;
}

export interface WorkflowExecutionState {
  status: WorkflowExecutionStatus;
  progress: WorkflowExecutionProgress;
}

export const ZERO_EXECUTION_STATE: WorkflowExecutionState = {
  status: "idle",
  progress: {
    currentIteration: 0,
    totalIterations: 0,
    currentWorkflow: null,
    currentWorkflowIndex: 0,
    totalWorkflows: 0,
  },
};

export interface WorkflowRunConfig {
  workflows: WorkflowType[];
  iterations: number;
}

export const DEFAULT_RUN_CONFIG: WorkflowRunConfig = {
  workflows: [],
  iterations: 1,
};
