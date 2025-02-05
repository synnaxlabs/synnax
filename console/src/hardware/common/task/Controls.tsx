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
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { type ReturnState } from "@/hardware/common/task/useDesiredState";
import { Layout } from "@/layout";

export interface ControlsProps {
  layoutKey: string;
  state: ReturnState;
  onStartStop: () => void;
  onConfigure: () => void;
  isConfiguring: boolean;
  isSnapshot: boolean;
}

const CONFIGURE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

export const Controls = ({
  state,
  onStartStop,
  layoutKey,
  onConfigure,
  isConfiguring: configuring,
  isSnapshot: snapshot = false,
}: ControlsProps) => {
  let content: ReactElement | null = null;
  if (state?.message != null)
    content = (
      <Status.Text variant={state?.variant ?? "info"}>{state?.message}</Status.Text>
    );
  if (snapshot)
    content = (
      <Status.Text.Centered variant="disabled" hideIcon>
        This task is a snapshot and cannot be modified or started.
      </Status.Text.Centered>
    );
  const isActive = Layout.useSelectActiveMosaicTabKey() === layoutKey;
  return (
    <Align.Space
      direction="x"
      className={CSS.B("task-controls")}
      justify="spaceBetween"
    >
      <Align.Space
        className={CSS.B("task-state")}
        direction="x"
        style={{
          borderRadius: "1rem",
          border: "var(--pluto-border)",
          padding: "2rem",
          width: "100%",
        }}
      >
        {content}
      </Align.Space>
      <Align.Space
        direction="x"
        bordered
        rounded
        style={{ padding: "2rem", borderRadius: "1rem" }}
        justify="end"
      >
        <Button.Icon
          loading={state.state === "loading"}
          disabled={state.state === "loading" || snapshot}
          onClick={onStartStop}
          variant="outlined"
        >
          {state.state === "running" ? <Icon.Pause /> : <Icon.Play />}
        </Button.Icon>
        <Button.Button
          loading={configuring}
          disabled={configuring || snapshot}
          onClick={onConfigure}
          triggers={isActive ? [CONFIGURE_TRIGGER] : undefined}
          tooltip={
            <Align.Space direction="x" align="center" size="small">
              <Triggers.Text shade={7} level="small" trigger={CONFIGURE_TRIGGER} />
              <Text.Text shade={7} level="small">
                To Configure
              </Text.Text>
            </Align.Space>
          }
        >
          Configure
        </Button.Button>
      </Align.Space>
    </Align.Space>
  );
};
