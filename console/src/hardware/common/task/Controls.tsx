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
import { type State } from "@/hardware/common/task/useState";
import { Layout } from "@/layout";

export interface ControlsProps {
  layoutKey: string;
  state: State;
  onStartStop: (command: "start" | "stop") => void;
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
  isSnapshot,
}: ControlsProps) => {
  const stateContent =
    state.message != null ? (
      <Status.Text variant={state.variant ?? "info"}>{state.message}</Status.Text>
    ) : isSnapshot ? (
      <Status.Text.Centered hideIcon variant="disabled">
        This task is a snapshot and cannot be modified or started.
      </Status.Text.Centered>
    ) : configured ? null : (
      <Status.Text.Centered variant="disabled" hideIcon>
        Task must be configured to start.
      </Status.Text.Centered>
    );
  const isLoading = state.status === "loading";
  const canConfigure = !isLoading && !isConfiguring && !isSnapshot;
  const canStartOrStop = !isLoading && !isConfiguring && !isSnapshot && configured;
  const hasTriggers =
    Layout.useSelectActiveMosaicTabKey() === layoutKey && canConfigure;
  const isRunning = state.status === "running";
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
            disabled={!canConfigure}
            loading={isConfiguring}
            onClick={onConfigure}
            size="medium"
            tooltip={
              hasTriggers ? (
                <Align.Space direction="x" align="center" size="small">
                  <Triggers.Text level="small" shade={7} trigger={CONFIGURE_TRIGGER} />
                  <Text.Text level="small" shade={7}>
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
            onClick={() => onStartStop(isRunning ? "stop" : "start")}
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
