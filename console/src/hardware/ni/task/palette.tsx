// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, task, UnexpectedError } from "@synnaxlabs/client";
import { Access, Icon } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { ANALOG_READ_LAYOUT } from "@/hardware/ni/task/AnalogRead";
import { ANALOG_WRITE_LAYOUT } from "@/hardware/ni/task/AnalogWrite";
import { COUNTER_READ_LAYOUT } from "@/hardware/ni/task/CounterRead";
import { DIGITAL_READ_LAYOUT } from "@/hardware/ni/task/DigitalRead";
import { DIGITAL_WRITE_LAYOUT } from "@/hardware/ni/task/DigitalWrite";
import { SCAN_SCHEMAS, SCAN_TYPE } from "@/hardware/ni/task/types";
import { Palette } from "@/palette";

const useVisible = () => Access.useUpdateGranted(task.TYPE_ONTOLOGY_ID);

export const CreateAnalogReadCommand = Palette.createSimpleCommand({
  key: "ni-create-analog-read-task",
  name: "Create an NI Analog Read Task",
  icon: <Icon.Logo.NI />,
  layout: ANALOG_READ_LAYOUT,
  useVisible,
});

export const CreateAnalogWriteCommand = Palette.createSimpleCommand({
  key: "ni-create-analog-write-task",
  name: "Create an NI Analog Write Task",
  icon: <Icon.Logo.NI />,
  layout: ANALOG_WRITE_LAYOUT,
  useVisible,
});

export const CreateCounterReadCommand = Palette.createSimpleCommand({
  key: "ni-create-counter-read-task",
  name: "Create an NI Counter Read Task",
  icon: <Icon.Logo.NI />,
  layout: COUNTER_READ_LAYOUT,
  useVisible,
});

export const CreateDigitalWriteCommand = Palette.createSimpleCommand({
  key: "ni-create-digital-write-task",
  name: "Create an NI Digital Write Task",
  icon: <Icon.Logo.NI />,
  layout: DIGITAL_WRITE_LAYOUT,
  useVisible,
});

export const CreateDigitalReadCommand = Palette.createSimpleCommand({
  key: "ni-create-digital-read-task",
  name: "Create an NI Digital Read Task",
  icon: <Icon.Logo.NI />,
  layout: DIGITAL_READ_LAYOUT,
  useVisible,
});

export const ToggleScannerCommand: Palette.Command = ({
  client,
  addStatus,
  handleError,
  ...listProps
}) => {
  const handleSelect = useCallback(
    () =>
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
        const scanTasks = await client.tasks.retrieve({
          types: [SCAN_TYPE],
          schemas: SCAN_SCHEMAS,
        });
        if (scanTasks.length === 0)
          throw new UnexpectedError("No NI device scanner found");
        const { config, payload } = scanTasks[0];
        const {
          config: { enabled },
        } = await client.tasks.create(
          {
            ...payload,
            config: { ...config, enabled: !config.enabled },
          },
          SCAN_SCHEMAS,
        );
        addStatus({
          variant: "success",
          message: `NI device scanning ${enabled ? "enabled" : "disabled"}`,
        });
      }, "Failed to toggle NI device scanner"),
    [client, addStatus, handleError],
  );
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Toggle the NI Device Scanner"
      icon={<Icon.Logo.NI />}
      onSelect={handleSelect}
    />
  );
};
ToggleScannerCommand.key = "ni-toggle-scan-task";
ToggleScannerCommand.commandName = "Toggle the NI Device Scanner";
ToggleScannerCommand.useVisible = useVisible;

export const COMMANDS = [
  CreateAnalogReadCommand,
  CreateAnalogWriteCommand,
  CreateCounterReadCommand,
  CreateDigitalWriteCommand,
  CreateDigitalReadCommand,
  ToggleScannerCommand,
];
