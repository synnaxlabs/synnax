// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align, Button, Icon, Status, Text, Triggers } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { CSS } from "@/css";
import {
  LOADING_STATUS,
  RUNNING_STATUS,
  START_COMMAND,
  type StartOrStopCommand,
  STOP_COMMAND,
} from "@/hardware/common/task/types";
import { type State } from "@/hardware/common/task/useState";
import { Layout } from "@/layout";

export interface ControlsProps extends Align.SpaceProps {
  layoutKey: string;
  state: State;
  onStartStop: (command: StartOrStopCommand) => void;
  onConfigure: () => void;
  isConfiguring: boolean;
  isSnapshot: boolean;
  hasBeenConfigured: boolean;
}

const CONFIGURE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

export const Controls = ({
  state: { message, status, variant },
  onStartStop,
  layoutKey,
  onConfigure,
  hasBeenConfigured,
  isConfiguring,
  isSnapshot,
  ...props
}: ControlsProps) => {
  const content = isSnapshot ? (
    <Status.Text.Centered hideIcon variant="disabled">
      This task is a snapshot and cannot be modified or started.
    </Status.Text.Centered>
  ) : message != null ? (
    <Status.Text variant={variant}>{message}</Status.Text>
  ) : isConfiguring ? (
    <Status.Text.Centered variant="loading">Configuring...</Status.Text.Centered>
  ) : !hasBeenConfigured ? (
    <Status.Text.Centered hideIcon variant="disabled">
      Task must be configured to start.
    </Status.Text.Centered>
  ) : null;
  const isLoading = status === LOADING_STATUS;
  const canConfigure = !isLoading && !isConfiguring && !isSnapshot;
  const canStartOrStop =
    !isLoading && !isConfiguring && !isSnapshot && hasBeenConfigured;
  const hasTriggers =
    Layout.useSelectActiveMosaicTabKey() === layoutKey && canConfigure;
  const isRunning = status === RUNNING_STATUS;
  const handleStartStop = useCallback(
    () => onStartStop(isRunning ? STOP_COMMAND : START_COMMAND),
    [isRunning, onStartStop],
  );
  return (
    <Align.Space
      className={CSS.B("task-controls")}
      x
      justify="spaceBetween"
      empty
      bordered
      {...props}
    >
      <Align.Space className={CSS.B("task-state")} x>
        {content}
      </Align.Space>
      {!isSnapshot && (
        <Align.Space align="center" x justify="end">
          <Button.Button
            disabled={!canConfigure}
            loading={isConfiguring}
            onClick={onConfigure}
            size="medium"
            tooltip={
              hasTriggers ? (
                <Align.Space x align="center" size="small">
                  <Triggers.Text level="small" shade={11} trigger={CONFIGURE_TRIGGER} />
                  <Text.Text level="small" shade={11}>
                    To Configure
                  </Text.Text>
                </Align.Space>
              ) : undefined
            }
            triggers={hasTriggers ? [CONFIGURE_TRIGGER] : undefined}
            variant="outlined"
          >
            Configure
          </Button.Button>
          <Button.Icon
            disabled={!canStartOrStop}
            loading={isLoading}
            onClick={handleStartStop}
            size="medium"
            variant="filled"
          >
            {isRunning ? <Icon.Pause /> : <Icon.Play />}
          </Button.Icon>
        </Align.Space>
      )}
    </Align.Space>
  );
};
