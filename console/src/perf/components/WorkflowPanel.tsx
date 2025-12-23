// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/perf/components/WorkflowPanel.css";

import { Button, Flex, Icon, Progress, Text } from "@synnaxlabs/pluto";
import { memo, type ReactElement, useCallback, useEffect, useRef, useState } from "react";

import { WorkflowConfigInputs } from "@/perf/components/WorkflowConfigInputs";
import { WorkflowSelect } from "@/perf/components/WorkflowSelect";
import { useWorkflowExecution } from "@/perf/hooks/useWorkflowExecution";
import { formatTime } from "@/perf/utils/formatting";
import { DEFAULT_WORKFLOW_CONFIG, type WorkflowConfig } from "@/perf/workflows/types";

const WorkflowPanelImpl = (): ReactElement => {
  const { state, start, cancel } = useWorkflowExecution();
  const [config, setConfig] = useState<WorkflowConfig>(DEFAULT_WORKFLOW_CONFIG);
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

  const handleWorkflowsChange = useCallback((workflows: string[]) => {
    setConfig((prev) => ({ ...prev, workflows }));
  }, []);

  const handleRun = useCallback(() => {
    if (config.workflows.length === 0) return;
    start(config);
  }, [config, start]);

  const isCancelled = state.status === "cancelled";
  const canRun = config.workflows.length > 0 && !isRunning;

  const progressPercent =
    state.progress.totalIterations > 0
      ? (state.progress.currentIteration / state.progress.totalIterations) * 100
      : 0;

  return (
    <Flex.Box y className="console-perf-workflow-panel">
      <Text.Text level="small" weight={500}>
        Macros
      </Text.Text>

      <WorkflowSelect
        value={config.workflows}
        onChange={handleWorkflowsChange}
        disabled={isRunning}
      />

      <WorkflowConfigInputs config={config} onChange={setConfig} disabled={isRunning} />

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

export const WorkflowPanel = memo(WorkflowPanelImpl);
