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
import { type MacroExecutionState, ZERO_EXECUTION_STATE } from "@/perf/macros/execution";
import { MacroRunner } from "@/perf/macros/runner";
import { type MacroConfig, type MacroContext } from "@/perf/macros/types";
import { type RootStore } from "@/store";

export interface UseMacroExecutionReturn {
  state: MacroExecutionState;
  start: (config: MacroConfig) => void;
  cancel: () => void;
}

export const useMacroExecution = (): UseMacroExecutionReturn => {
  const dispatch = useDispatch();
  const store = useStore() as RootStore;
  const placer = usePlacer();
  const client = Synnax.use();
  const runnerRef = useRef<MacroRunner | null>(null);
  const [state, setState] = useState<MacroExecutionState>(ZERO_EXECUTION_STATE);

  const start = useCallback(
    (config: MacroConfig) => {
      if (state.status === "running") return;

      const context: MacroContext = {
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
          currentMacro: config.macros[0] ?? null,
          currentMacroIndex: 0,
          totalMacros: config.macros.length,
        },
      });

      const runner = new MacroRunner(context, config, {
        onMacroComplete: () => {
          setState((prev) => {
            const nextMacroIndex = prev.progress.currentMacroIndex + 1;
            const isLastMacro = nextMacroIndex >= config.macros.length;

            if (isLastMacro) {
              const nextIteration = prev.progress.currentIteration + 1;
              if (nextIteration >= config.iterations)
                return { ...prev, status: "idle", progress: ZERO_EXECUTION_STATE.progress };
              
              return {
                ...prev,
                progress: {
                  ...prev.progress,
                  currentIteration: nextIteration,
                  currentMacroIndex: 0,
                  currentMacro: config.macros[0] ?? null,
                },
              };
            }

            return {
              ...prev,
              progress: {
                ...prev.progress,
                currentMacroIndex: nextMacroIndex,
                currentMacro: config.macros[nextMacroIndex] ?? null,
              },
            };
          });
        },
      });

      runnerRef.current = runner;
      void runner
        .run()
        .catch((e) => console.error("Macro runner error:", e))
        .finally(() => {
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
