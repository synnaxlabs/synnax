// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { type Flex, type Flux, Form } from "@synnaxlabs/pluto";
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
  formStatus: Flux.Result<undefined>["status"];
  /** Runs the deploy pipeline: configure, save the row, issue start. */
  onDeploy: () => void;
  /** Issues a stop command to the live instance. */
  onStop: () => void;
}

/**
 * Task controls component that wires up the presentational controls
 * with task-specific data from Form context.
 */
export const Controls = ({ onDeploy, onStop, formStatus, ...props }: ControlsProps) => {
  const taskStatus = useStatus();
  const isSnapshot = Form.useFieldValue<boolean>("snapshot");
  const key = useKey();
  const drifted = useDrifted();

  const [expanded, setExpanded] = useState(false);

  // Compute effective status (form errors take precedence)
  let effectiveStatus: status.Status = taskStatus;
  if (formStatus.variant !== "success") effectiveStatus = formStatus;

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

  const formInvalid = formStatus.variant !== "success";
  return (
    <Frame expanded={expanded} onContract={handleContract} {...props}>
      <Status status={effectiveStatus} expanded={expanded} onToggle={handleToggle} />
      {!isSnapshot && (
        <Actions>
          {drifted && (
            <RedeployButton onClick={handleRedeploy} disabled={formInvalid} />
          )}
          <StartStopButton
            running={running}
            onClick={handleStartStop}
            disabled={formInvalid}
            statusVariant={status.keepVariants(taskStatus.variant, "loading")}
          />
        </Actions>
      )}
    </Frame>
  );
};
