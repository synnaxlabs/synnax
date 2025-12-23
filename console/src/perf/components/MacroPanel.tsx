// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/perf/components/MacroPanel.css";

import { Button, Flex, Icon, Progress, Text } from "@synnaxlabs/pluto";
import { memo, type ReactElement, useCallback, useEffect, useRef, useState } from "react";

import { MacroConfigInputs } from "@/perf/components/MacroConfigInputs";
import { MacroSelect } from "@/perf/components/MacroSelect";
import { useMacroExecution } from "@/perf/hooks/useMacroExecution";
import { DEFAULT_MACRO_CONFIG, type MacroConfig } from "@/perf/macros/types";
import { formatTime } from "@/perf/utils/formatting";

const MacroPanelImpl = (): ReactElement => {
  const { state, start, cancel } = useMacroExecution();
  const [config, setConfig] = useState<MacroConfig>(DEFAULT_MACRO_CONFIG);
  const [elapsedSeconds, setElapsedSeconds] = useState(0);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const isRunning = state.status === "running";

  useEffect(() => {
    if (isRunning) {
      setElapsedSeconds(0);
      timerRef.current = setInterval(() => {
        setElapsedSeconds((prev) => prev + 1);
      }, 1000);
    } else if (timerRef.current != null) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }
    return () => {
      if (timerRef.current != null) clearInterval(timerRef.current);
    };
  }, [isRunning]);

  const handleMacrosChange = useCallback((macros: string[]) => {
    setConfig((prev) => ({ ...prev, macros }));
  }, []);

  const handleRun = useCallback(() => {
    if (config.macros.length === 0) return;
    start(config);
  }, [config, start]);

  const isCancelled = state.status === "cancelled";
  const canRun = config.macros.length > 0 && !isRunning;

  const progressPercent =
    state.progress.totalIterations > 0
      ? (state.progress.currentIteration / state.progress.totalIterations) * 100
      : 0;

  return (
    <Flex.Box y className="console-perf-macro-panel">
      <Text.Text level="small" weight={500}>
        Macros
      </Text.Text>

      <MacroSelect
        value={config.macros}
        onChange={handleMacrosChange}
        disabled={isRunning}
      />

      <MacroConfigInputs config={config} onChange={setConfig} disabled={isRunning} />

      <Flex.Box x gap="small">
        {isRunning ? (
          <Button.Button variant="outlined" size="small" onClick={cancel}>
            <Icon.Pause />
            Cancel
          </Button.Button>
        ) : (
          <Button.Button
            variant="filled"
            size="small"
            onClick={handleRun}
            disabled={!canRun}
          >
            <Icon.Play />
            Run
          </Button.Button>
        )}
      </Flex.Box>

      {(isRunning || isCancelled) && (
        <Flex.Box y gap="small">
          <Progress.Progress value={progressPercent} />
          <Flex.Box x justify="between" align="center">
            <Text.Text level="small">
              {isCancelled
                ? "Cancelled"
                : `Iteration ${state.progress.currentIteration + 1} / ${state.progress.totalIterations}`}
            </Text.Text>
            <Text.Text level="small">{formatTime(elapsedSeconds)}</Text.Text>
          </Flex.Box>
        </Flex.Box>
      )}
    </Flex.Box>
  );
};

export const MacroPanel = memo(MacroPanelImpl);
