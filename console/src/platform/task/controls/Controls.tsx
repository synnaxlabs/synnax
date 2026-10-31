// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { type Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Status } from "@synnaxlabs/lyra/status";

import { Framer } from "@/platform/framer";
import { Bar } from "@/platform/task/controls/Bar";
import { useDrifted } from "@/platform/task/useDrifted";
import { useKey } from "@/platform/task/useKey";
import { useStatus } from "@/platform/task/useStatus";
import { Session } from "@/session";

export interface ControlsProps extends Flex.BoxProps {
  /** The status of the last save of the task. A failed save shows in place. */
  saveStatus: status.Status;
  /** Runs the deploy pipeline: configure, save the row, issue start. */
  onDeploy: () => void;
  /** Issues a stop command to the live instance. */
  onStop: () => void;
}

/** Task controls bar wired to the surrounding task Form context. */
export const Controls = ({ saveStatus, onDeploy, onStop, ...props }: ControlsProps) => {
  const taskStatus = useStatus();
  const saveFailed = saveStatus.variant === "error";
  const visible = Session.Panel.useSelectIsTabVisible();
  // A failed save replaces the task status, so its key is placed before the save fails.
  Status.usePlace(saveStatus.key, visible);
  Status.usePlace(taskStatus.key, visible && !saveFailed);
  const isSnapshot = Form.useFieldValue<boolean>("snapshot");
  const key = useKey();
  const drifted = useDrifted();
  const canControl = Framer.useCanCommand();

  const handleDeploy = () => {
    if (key == null) return;
    onDeploy();
  };
  const handleStop = () => {
    if (key == null) return;
    onStop();
  };

  return (
    <Bar
      status={saveFailed ? saveStatus : taskStatus}
      running={taskStatus.details.running}
      drifted={drifted}
      hideActions={isSnapshot || !canControl}
      startStopVariant={status.keepVariants(taskStatus.variant, "loading")}
      onDeploy={handleDeploy}
      onStop={handleStop}
      {...props}
    />
  );
};
