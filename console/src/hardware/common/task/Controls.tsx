// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Button,
  Flex,
  type Flux,
  Form,
  Icon,
  Status,
  Synnax,
  Text,
  Triggers,
} from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";
import { useCallback } from "react";

import { CSS } from "@/css";
import { useKey } from "@/hardware/common/task/useKey";
import { useStatus } from "@/hardware/common/task/useStatus";
import { Layout } from "@/layout";

export interface ControlsProps extends Flex.BoxProps {
  layoutKey: string;
  formStatus: Flux.Result<undefined>["status"];
  onConfigure: () => void;
}

const CONFIGURE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

export const Controls = ({
  layoutKey,
  onConfigure,
  formStatus,
  ...props
}: ControlsProps) => {
  const taskStatus = useStatus();
  const isSnapshot = Form.useFieldValue<boolean>("snapshot");
  const handleError = Status.useErrorHandler();
  let stat: status.Status = taskStatus;
  if (formStatus.variant !== "success") stat = formStatus;
  const hasTriggers = Layout.useSelectActiveMosaicTabKeyAndNotBlurred() != null;
  const client = Synnax.use();
  const key = useKey();
  const handleStartStop = useCallback(() => {
    if (key == null) return;
    const command = taskStatus.details.running ? "stop" : "start";
    handleError(
      async () => await client?.tasks.executeCommand({ task: key, type: command }),
      `Failed to ${command} task`,
    );
  }, [taskStatus]);
  console.log(taskStatus.message ?? "No description");
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
        <Status.Summary status={stat} justify="center" align="center" center={false} />
      </Flex.Box>
      {!isSnapshot && (
        <Flex.Box align="center" x justify="end">
          <Button.Button
            onClick={onConfigure}
            status={status.keepVariants(formStatus.variant, ["loading", "disabled"])}
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
            status={status.keepVariants(taskStatus.variant, "loading")}
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
