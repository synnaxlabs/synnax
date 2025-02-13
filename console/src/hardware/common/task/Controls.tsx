// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Align, Button, Status, Text, Triggers } from "@synnaxlabs/pluto";

import { CSS } from "@/css";
import { type ReturnState } from "@/hardware/common/task/useState";
import { Layout } from "@/layout";

export interface ControlsProps {
  layoutKey: string;
  state: ReturnState;
  onStartStop: () => void;
  onConfigure: () => void;
  isConfiguring: boolean;
  isSnapshot: boolean;
  configured: boolean;
}

const CONFIGURE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

export const Controls = ({
  state,
  onStartStop,
  layoutKey,
  onConfigure,
  configured,
  isConfiguring,
  isSnapshot = false,
}: ControlsProps) => {
  let stateContent = (
    <Status.Text.Centered variant="disabled" hideIcon>
      Task must be configured to start.
    </Status.Text.Centered>
  );
  if (state.message != null)
    stateContent = (
      <Status.Text variant={state.variant ?? "info"}>{state.message}</Status.Text>
    );
  else if (isSnapshot)
    stateContent = (
      <Status.Text.Centered hideIcon variant="disabled">
        This task is a snapshot and cannot be modified or started.
      </Status.Text.Centered>
    );

  const isActive = Layout.useSelectActiveMosaicTabKey() === layoutKey;
  const isLoading = state.state === "loading";
  const configureDisabled = isLoading || isConfiguring || isSnapshot;
  const startStopDisabled = isLoading || isConfiguring || isSnapshot || !configured;
  return (
    <Align.Space
      className={CSS.B("task-controls")}
      direction="x"
      justify="spaceBetween"
      empty
      bordered
    >
      <Align.Space className={CSS.B("task-state")} direction="x">
        {stateContent}
      </Align.Space>
      {!isSnapshot && (
        <Align.Space align="center" direction="x" justify="end">
          <Button.Button
            disabled={configureDisabled}
            loading={isConfiguring}
            onClick={onConfigure}
            size="medium"
            tooltip={
              <Align.Space direction="x" align="center" size="small">
                <Triggers.Text level="small" shade={7} trigger={CONFIGURE_TRIGGER} />
                <Text.Text level="small" shade={7}>
                  To Configure
                </Text.Text>
              </Align.Space>
            }
            triggers={isActive ? [CONFIGURE_TRIGGER] : undefined}
            variant="outlined"
          >
            Configure
          </Button.Button>
          <Button.Icon
            disabled={startStopDisabled}
            loading={isLoading}
            onClick={onStartStop}
            size="medium"
            variant="filled"
          >
            {state.state === "running" ? <Icon.Pause /> : <Icon.Play />}
          </Button.Icon>
        </Align.Space>
      )}
    </Align.Space>
  );
};
