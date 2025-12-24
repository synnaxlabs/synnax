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
        onMacroStart: (macroType, macroIndex, iteration) => {
          setState((prev) => ({
            ...prev,
            progress: {
              ...prev.progress,
              currentIteration: iteration,
              currentMacroIndex: macroIndex,
              currentMacro: macroType,
            },
          }));
        },
      });

      runnerRef.current = runner;
      void runner
        .run()
        .catch((e) => console.error("Macro runner error:", e))
        .finally(() => {
          setState((prev) =>
            prev.status === "running" ? { ...prev, status: "idle" } : prev,
          );
          runnerRef.current = null;
        });
    },
    [state.status, store, dispatch, placer, client],
  );

  const cancel = useCallback(() => {
    runnerRef.current?.stop();
    setState((prev) => ({ ...prev, status: "cancelled" }));
    setTimeout(() => {
      setState((prev) => ({ ...prev, status: "idle" }));
    }, 1000);
  }, []);

  return { state, start, cancel };
};
