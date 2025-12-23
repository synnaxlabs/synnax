// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax } from "@synnaxlabs/pluto";
import { useCallback, useRef, useState } from "react";
import { useDispatch, useStore } from "react-redux";

import { usePlacer } from "@/layout/usePlacer";
import { type RootStore } from "@/store";
import {
  type WorkflowExecutionState,
  ZERO_EXECUTION_STATE,
} from "@/perf/workflows/execution";
import { WorkflowRunner } from "@/perf/workflows/runner";
import { type WorkflowConfig, type WorkflowContext } from "@/perf/workflows/types";

export interface UseWorkflowExecutionReturn {
  state: WorkflowExecutionState;
  start: (config: WorkflowConfig) => void;
  cancel: () => void;
}

export const useWorkflowExecution = (): UseWorkflowExecutionReturn => {
  const dispatch = useDispatch();
  const store = useStore() as RootStore;
  const placer = usePlacer();
  const client = Synnax.use();
  const runnerRef = useRef<WorkflowRunner | null>(null);
  const [state, setState] = useState<WorkflowExecutionState>(ZERO_EXECUTION_STATE);

  const start = useCallback(
    (config: WorkflowConfig) => {
      if (state.status === "running") return;

      const context: WorkflowContext = {
        store,
        dispatch,
        placer,
        client,
        createdLayoutKeys: [],
        availableChannelKeys: [],
      };

      setState({
        status: "running",
        progress: {
          currentIteration: 0,
          totalIterations: config.iterations,
          currentWorkflow: config.workflows[0] ?? null,
          currentWorkflowIndex: 0,
          totalWorkflows: config.workflows.length,
        },
      });

      const runner = new WorkflowRunner(context, config, {
        onWorkflowComplete: (result) => {
          setState((prev) => {
            const nextWorkflowIndex = prev.progress.currentWorkflowIndex + 1;
            const isLastWorkflow = nextWorkflowIndex >= config.workflows.length;

            if (isLastWorkflow) {
              const nextIteration = prev.progress.currentIteration + 1;
              if (nextIteration > config.iterations) {
                return { ...prev, status: "idle", progress: ZERO_EXECUTION_STATE.progress };
              }
              return {
                ...prev,
                progress: {
                  ...prev.progress,
                  currentIteration: nextIteration,
                  currentWorkflowIndex: 0,
                  currentWorkflow: config.workflows[0] ?? null,
                },
              };
            }

            return {
              ...prev,
              progress: {
                ...prev.progress,
                currentWorkflowIndex: nextWorkflowIndex,
                currentWorkflow: config.workflows[nextWorkflowIndex] ?? null,
              },
            };
          });
        },
      });

      runnerRef.current = runner;
      runner.run().finally(() => {
        setState((prev) =>
          prev.status === "running"
            ? { ...prev, status: "idle", progress: ZERO_EXECUTION_STATE.progress }
            : prev,
        );
        runnerRef.current = null;
      });
    },
    [state.status, store, dispatch, placer, client],
  );

  const cancel = useCallback(() => {
    runnerRef.current?.stop();
    setState({ status: "cancelled", progress: ZERO_EXECUTION_STATE.progress });
    setTimeout(() => {
      setState(ZERO_EXECUTION_STATE);
    }, 1000);
  }, []);

  return { state, start, cancel };
};
