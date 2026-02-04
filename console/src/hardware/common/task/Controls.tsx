// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Flex, type Flux, Form, Status, Synnax } from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";
import { useCallback, useState } from "react";

import { Controls as ControlsNS } from "@/hardware/common/task/task-controls";
import { useKey } from "@/hardware/common/task/useKey";
import { useStatus } from "@/hardware/common/task/useStatus";
import { Layout } from "@/layout";

export interface ControlsProps extends Flex.BoxProps {
  layoutKey: string;
  formStatus: Flux.Result<undefined>["status"];
  onConfigure: () => void;
}

/**
 * Task controls component that wires up the presentational controls
 * with task-specific data from Form context.
 */
export const Controls = ({
  layoutKey,
  onConfigure,
  formStatus,
  ...props
}: ControlsProps) => {
  const taskStatus = useStatus();
  const isSnapshot = Form.useFieldValue<boolean>("snapshot");
  const handleError = Status.useErrorHandler();
  const client = Synnax.use();
  const key = useKey();
  const hasTriggers = Layout.useSelectActiveMosaicTabKeyAndNotBlurred() != null;

  const [expanded, setExpanded] = useState(false);
  const [hovered, setHovered] = useState(false);

  // Compute effective status (form errors take precedence)
  let effectiveStatus: status.Status = taskStatus;
  if (formStatus.variant !== "success") effectiveStatus = formStatus;

  const handleStartStop = useCallback(() => {
    if (key == null) return;
    const command = taskStatus.details.running ? "stop" : "start";
    handleError(
      async () => await client?.tasks.executeCommand({ task: key, type: command }),
      `Failed to ${command} task`,
    );
  }, [taskStatus, key, client, handleError]);

  const handleToggle = useCallback(() => {
    setExpanded((prev) => !prev);
    setHovered(false);
  }, []);

  return (
    <ControlsNS.Frame expanded={expanded} hovered={hovered} {...props}>
      <ControlsNS.ExpandableStatus
        status={effectiveStatus}
        expanded={expanded}
        hovered={hovered}
        onToggle={handleToggle}
        onHoverChange={setHovered}
      />
      {!isSnapshot && (
        <ControlsNS.Actions>
          <ControlsNS.ConfigureButton
            onClick={onConfigure}
            showTrigger={hasTriggers}
            statusVariant={status.keepVariants(formStatus.variant, [
              "loading",
              "disabled",
            ])}
          />
          <ControlsNS.StartStopButton
            running={taskStatus.details.running}
            onClick={handleStartStop}
            disabled={formStatus.variant !== "success"}
            statusVariant={status.keepVariants(taskStatus.variant, "loading")}
          />
        </ControlsNS.Actions>
      )}
    </ControlsNS.Frame>
  );
};
