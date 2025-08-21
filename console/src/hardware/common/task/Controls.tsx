// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import {
  Button,
  Flex,
  Flux,
  Form,
  Icon,
  Status,
  Synnax,
  Text,
  Triggers,
} from "@synnaxlabs/pluto";
import { type z } from "zod";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { status } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useKey, useStatus } from "@/hardware/common/task/Form";

export interface ControlsProps extends Flex.BoxProps {
  layoutKey: string;
  formStatus: Flux.Result<undefined>["status"];
  onConfigure: () => void;
}

const CONFIGURE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

export const Controls = <StatusData extends z.ZodType = z.ZodType>({
  layoutKey,
  onConfigure,
  formStatus,
  ...props
}: ControlsProps) => {
  const taskStatus = useStatus();
  const isSnapshot = Form.useFieldValue<boolean>("snapshot");
  let stat: status.Status = taskStatus;
  if (formStatus.variant !== "success") stat = formStatus;
  const hasTriggers = Layout.useSelectActiveMosaicTabKey() === layoutKey;
  const client = Synnax.use();
  const key = useKey();
  const handleStartStop = useCallback(async () => {
    if (key == null) return;
    await client?.hardware.tasks.executeCommand(
      key,
      taskStatus.details.running ? "stop" : "start",
    );
  }, [taskStatus]);
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
        <Status.Summary variant={stat.variant}>{stat.message}</Status.Summary>
      </Flex.Box>
      {!isSnapshot && (
        <Flex.Box align="center" x justify="end">
          <Button.Button
            onClick={onConfigure}
            status={status.filterVariant(formStatus.variant, ["loading", "disabled"])}
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
            disabled={formStatus.variant !== "success"}
            status={status.filterVariant(taskStatus.variant, "loading")}
            onClick={handleStartStop}
            size="medium"
            variant="filled"
          >
            {taskStatus.details.running ? <Icon.Pause /> : <Icon.Play />}
          </Button.Button>
        </Flex.Box>
      )}
    </Flex.Box>
  );
};
