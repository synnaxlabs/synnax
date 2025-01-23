// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Button, Status, Text, Triggers } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Layout } from "@/layout";

export interface ControlsProps {
  layoutKey: string;
  onStartStop: () => void;
  startingOrStopping: boolean;
  onConfigure: () => void;
  configuring: boolean;
  snapshot?: boolean;
  state?: task.State<{ running?: boolean; message?: string }>;
}

const CONFIGURE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

export const Controls = ({
  state,
  onStartStop,
  layoutKey,
  startingOrStopping,
  onConfigure,
  configuring,
  snapshot = false,
}: ControlsProps) => {
  let content: ReactElement | null = null;
  if (state?.details?.message != null)
    content = (
      <Status.Text variant={state?.variant as Status.Variant}>
        {state?.details?.message}
      </Status.Text>
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
          loading={startingOrStopping}
          disabled={startingOrStopping || state == null || snapshot}
          onClick={onStartStop}
          variant="outlined"
        >
          {state?.details?.running === true ? <Icon.Pause /> : <Icon.Play />}
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
