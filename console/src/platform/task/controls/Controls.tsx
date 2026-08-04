// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { type Flex, type Flux, Form } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Bar } from "@/platform/task/controls/Bar";
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

/** Task controls bar wired to the surrounding task Form context. */
export const Controls = ({ onDeploy, onStop, formStatus, ...props }: ControlsProps) => {
  const taskStatus = useStatus();
  const isSnapshot = Form.useFieldValue<boolean>("snapshot");
  const key = useKey();
  const drifted = useDrifted();

  // Form errors take precedence over the task status.
  let effectiveStatus: status.Status = taskStatus;
  if (formStatus.variant !== "success") effectiveStatus = formStatus;

  const handleDeploy = useCallback(() => {
    if (key == null) return;
    onDeploy();
  }, [key, onDeploy]);
  const handleStop = useCallback(() => {
    if (key == null) return;
    onStop();
  }, [key, onStop]);

  return (
    <Bar
      status={effectiveStatus}
      running={taskStatus.details.running}
      drifted={drifted}
      snapshot={isSnapshot}
      disabled={formStatus.variant !== "success"}
      onDeploy={handleDeploy}
      onStop={handleStop}
      {...props}
    />
  );
};
