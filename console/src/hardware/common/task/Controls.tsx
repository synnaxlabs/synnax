// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { Button, Flex, Icon, Status, Text, Triggers } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { type z } from "zod";

import { CSS } from "@/css";
import { type Command } from "@/hardware/common/task/types";
import { Layout } from "@/layout";

export interface ControlsProps<StatusData extends z.ZodType = z.ZodType>
  extends Flex.BoxProps {
  layoutKey: string;
  status: task.Status<StatusData>;
  onCommand: (command: Command) => void;
  onConfigure: () => void;
  isConfiguring: boolean;
  isSnapshot: boolean;
  hasBeenConfigured: boolean;
}

const CONFIGURE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

export const Controls = <StatusData extends z.ZodType = z.ZodType>({
  status,
  onCommand,
  layoutKey,
  onConfigure,
  hasBeenConfigured,
  isConfiguring,
  isSnapshot,
  ...props
}: ControlsProps<StatusData>) => {
  const {
    message,
    variant,
    details: { running },
  } = status ?? {};
  const content = isSnapshot ? (
    <Status.Text center hideIcon variant="disabled">
      This task is a snapshot and cannot be modified or started.
    </Status.Text>
  ) : message != null ? (
    <Status.Text variant={variant}>{message}</Status.Text>
  ) : isConfiguring ? (
    <Status.Text center variant="loading">
      Configuring...
    </Status.Text>
  ) : !hasBeenConfigured ? (
    <Status.Text center hideIcon variant="disabled">
      Task must be configured to start.
    </Status.Text>
  ) : null;
  const isLoading = variant === "loading";
  const canConfigure = !isLoading && !isConfiguring && !isSnapshot;
  const canStartOrStop =
    !isLoading && !isConfiguring && !isSnapshot && hasBeenConfigured;
  const hasTriggers =
    Layout.useSelectActiveMosaicTabKey() === layoutKey && canConfigure;
  const handleStartStop = useCallback(
    () => onCommand(running ? "stop" : "start"),
    [running, onCommand],
  );
  if (isConfiguring) status.variant = "loading";
  return (
    <Flex.Box
      className={CSS.B("task-controls")}
      x
      justify="between"
      empty
      bordered
      {...props}
    >
      <Flex.Box className={CSS.B("task-state")} x>
        {content}
      </Flex.Box>
      {!isSnapshot && (
        <Flex.Box align="center" x justify="end">
          <Button.Button
            disabled={!canConfigure || status.variant === "loading"}
            onClick={onConfigure}
            size="medium"
            tooltip={
              hasTriggers ? (
                <Flex.Box x align="center" gap="small">
                  <Triggers.Text level="small" trigger={CONFIGURE_TRIGGER} />
                  <Text.Text level="small">To Configure</Text.Text>
                </Flex.Box>
              ) : undefined
            }
            trigger={hasTriggers ? CONFIGURE_TRIGGER : undefined}
            variant="outlined"
          >
            Configure
          </Button.Button>
          <Button.Button
            disabled={!canStartOrStop}
            status={status.variant}
            onClick={handleStartStop}
            size="medium"
            variant="filled"
          >
            {running ? <Icon.Pause /> : <Icon.Play />}
          </Button.Button>
        </Flex.Box>
      )}
    </Flex.Box>
  );
};
