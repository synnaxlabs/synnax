// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/perf/components/MacroPanel.css";
import "@/perf/components/MetricRow.css";

import { Button, Icon, Progress, Text } from "@synnaxlabs/pluto";
import {
  memo,
  type ReactElement,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

import { MacroConfigInputs } from "@/perf/components/MacroConfigInputs";
import { MacroSelect } from "@/perf/components/MacroSelect";
import { Section } from "@/perf/components/Section";
import { useMacroExecution } from "@/perf/hooks/useMacroExecution";
import { useTimer } from "@/perf/hooks/useTimer";
import { DEFAULT_MACRO_CONFIG, type MacroConfig } from "@/perf/macros/types";
import { formatTime } from "@/perf/utils/formatting";

const MacroPanelImpl = (): ReactElement => {
  const { state, start, cancel } = useMacroExecution();
  const [config, setConfig] = useState<MacroConfig>(DEFAULT_MACRO_CONFIG);
  const isRunning = state.status === "running";
  const runningSeconds = useTimer(isRunning);
  const finalSecondsRef = useRef(0);

  // Capture elapsed time while running so we preserve it when complete
  useEffect(() => {
    if (isRunning) finalSecondsRef.current = runningSeconds;
  }, [isRunning, runningSeconds]);

  const elapsedSeconds = isRunning ? runningSeconds : finalSecondsRef.current;
  const hasRun = !isRunning && state.progress.totalIterations > 0;
  const isComplete =
    hasRun && state.progress.currentIteration + 1 >= state.progress.totalIterations;
  const isStopped = hasRun && !isComplete;

  const handleMacrosChange = useCallback((macros: string[]) => {
    setConfig((prev) => ({ ...prev, macros }));
  }, []);

  const handleRun = useCallback(() => {
    if (config.macros.length === 0) return;
    start(config);
  }, [config, start]);

  const canRun = config.macros.length > 0 && !isRunning;

  const progressPercent =
    state.progress.totalIterations > 0 && (isRunning || hasRun)
      ? ((state.progress.currentIteration + 1) / state.progress.totalIterations) * 100
      : 0;

  const actionButton = isRunning ? (
    <Button.Button variant="outlined" size="tiny" onClick={cancel}>
      <Icon.Pause />
      Stop
    </Button.Button>
  ) : (
    <Button.Button variant="filled" size="tiny" onClick={handleRun} disabled={!canRun}>
      <Icon.Play />
      Run
    </Button.Button>
  );

  const getStatusText = (): string => {
    if (isRunning)
      return `Iteration ${state.progress.currentIteration + 1} / ${state.progress.totalIterations}`;
    if (isComplete) return `Complete (${state.progress.totalIterations} iterations)`;
    if (isStopped)
      return `Stopped (${state.progress.currentIteration + 1} / ${state.progress.totalIterations})`;
    return "Idle";
  };

  const progressBar = (
    <div className="console-perf-macro-progress">
      <Progress.Progress value={progressPercent} />
      <div className="console-perf-macro-progress-overlay">
        <Text.Text
          level="small"
          className={isRunning || hasRun ? undefined : "console-perf-row-label"}
        >
          {getStatusText()}
        </Text.Text>
        <Text.Text level="small">
          {isRunning || hasRun ? formatTime(elapsedSeconds) : ""}
        </Text.Text>
      </div>
    </div>
  );

  return (
    <Section title="Macros" actions={actionButton} subheader={progressBar}>
      <MacroSelect
        value={config.macros}
        onChange={handleMacrosChange}
        disabled={isRunning}
      />
      <MacroConfigInputs config={config} onChange={setConfig} disabled={isRunning} />
    </Section>
  );
};

export const MacroPanel = memo(MacroPanelImpl);
