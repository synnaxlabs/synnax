// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Flex,
  type Flux,
  Form,
  Status as BaseStatus,
  Synnax,
} from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";
import { useCallback, useState } from "react";

import { Actions } from "@/hardware/common/task/controls/Actions";
import { ConfigureButton } from "@/hardware/common/task/controls/ConfigureButton";
import { Frame } from "@/hardware/common/task/controls/Frame";
import { StartStopButton } from "@/hardware/common/task/controls/StartStopButton";
import { Status } from "@/hardware/common/task/controls/Status";
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
  const handleError = BaseStatus.useErrorHandler();
  const client = Synnax.use();
  const key = useKey();
  const hasTriggers = Layout.useSelectActiveMosaicTabKeyAndNotBlurred() != null;

  const [expanded, setExpanded] = useState(false);

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
  }, []);

  return (
    <Frame expanded={expanded} {...props}>
      <Status status={effectiveStatus} expanded={expanded} onToggle={handleToggle} />
      {!isSnapshot && (
        <Actions>
          <ConfigureButton
            onClick={onConfigure}
            showTrigger={hasTriggers}
            statusVariant={status.keepVariants(formStatus.variant, [
              "loading",
              "disabled",
            ])}
          />
          <StartStopButton
            running={taskStatus.details.running}
            onClick={handleStartStop}
            disabled={formStatus.variant !== "success"}
            statusVariant={status.keepVariants(taskStatus.variant, "loading")}
          />
        </Actions>
      )}
    </Frame>
  );
};
