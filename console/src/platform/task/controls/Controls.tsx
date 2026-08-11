// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { type Flex, Form } from "@synnaxlabs/pluto";
import { useCallback, useState } from "react";

import { Actions } from "@/platform/task/controls/Actions";
import { Frame } from "@/platform/task/controls/Frame";
import { RedeployButton } from "@/platform/task/controls/RedeployButton";
import { StartStopButton } from "@/platform/task/controls/StartStopButton";
import { Status } from "@/platform/task/controls/Status";
import { useDrifted } from "@/platform/task/useDrifted";
import { useKey } from "@/platform/task/useKey";
import { useStatus } from "@/platform/task/useStatus";

export interface ControlsProps extends Flex.BoxProps {
  /** Runs the deploy pipeline: configure, save the row, issue start. */
  onDeploy: () => void;
  /** Issues a stop command to the live instance. */
  onStop: () => void;
}

/**
 * Task controls component that wires up the presentational controls
 * with task-specific data from Form context.
 */
export const Controls = ({ onDeploy, onStop, ...props }: ControlsProps) => {
  const taskStatus = useStatus();
  const isSnapshot = Form.useFieldValue<boolean>("snapshot");
  const key = useKey();
  const drifted = useDrifted();

  const [expanded, setExpanded] = useState(false);

  const running = taskStatus.details.running;
  const handleStartStop = useCallback(() => {
    if (key == null) return;
    if (running) onStop();
    else onDeploy();
  }, [key, running, onStop, onDeploy]);

  const handleRedeploy = useCallback(() => {
    if (key == null) return;
    onDeploy();
  }, [key, onDeploy]);

  const handleToggle = useCallback(() => setExpanded((prev) => !prev), []);
  const handleContract = useCallback(() => setExpanded(false), []);

  return (
    <Frame expanded={expanded} onContract={handleContract} {...props}>
      <Status status={taskStatus} expanded={expanded} onToggle={handleToggle} />
      {!isSnapshot && (
        <Actions>
          <RedeployButton
            visible={drifted}
            onClick={handleRedeploy}
            disabled={taskStatus.variant === "loading"}
          />
          <StartStopButton
            running={running}
            onClick={handleStartStop}
            statusVariant={status.keepVariants(taskStatus.variant, "loading")}
          />
        </Actions>
      )}
    </Frame>
  );
};
