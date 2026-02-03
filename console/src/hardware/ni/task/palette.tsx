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
import {
  importAnalogRead,
  importAnalogWrite,
  importCounterRead,
  importDigitalRead,
  importDigitalWrite,
} from "@/hardware/ni/task/import";
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

export const ImportAnalogReadCommand: Palette.Command = ({
  placeLayout,
  handleError,
  store,
  client,
  fluxStore,
  ...listProps
}) => {
  const handleSelect = useCallback(
    () => importAnalogRead({ placeLayout, handleError, store, client, fluxStore }),
    [placeLayout, handleError, store, client, fluxStore],
  );
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Import NI Analog Read Task(s)"
      icon={<Icon.Logo.NI />}
      onSelect={handleSelect}
    />
  );
};
ImportAnalogReadCommand.key = "ni-import-analog-read-task";
ImportAnalogReadCommand.commandName = "Import NI Analog Read Task(s)";
ImportAnalogReadCommand.sortOrder = -1;
ImportAnalogReadCommand.useVisible = useVisible;

export const ImportAnalogWriteCommand: Palette.Command = ({
  placeLayout,
  handleError,
  store,
  client,
  fluxStore,
  ...listProps
}) => {
  const handleSelect = useCallback(
    () => importAnalogWrite({ placeLayout, handleError, store, client, fluxStore }),
    [placeLayout, handleError, store, client, fluxStore],
  );
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Import NI Analog Write Task(s)"
      icon={<Icon.Logo.NI />}
      onSelect={handleSelect}
    />
  );
};
ImportAnalogWriteCommand.key = "ni-import-analog-write-task";
ImportAnalogWriteCommand.commandName = "Import NI Analog Write Task(s)";
ImportAnalogWriteCommand.sortOrder = -1;
ImportAnalogWriteCommand.useVisible = useVisible;

export const ImportCounterReadCommand: Palette.Command = ({
  placeLayout,
  handleError,
  store,
  client,
  fluxStore,
  ...listProps
}) => {
  const handleSelect = useCallback(
    () => importCounterRead({ placeLayout, handleError, store, client, fluxStore }),
    [placeLayout, handleError, store, client, fluxStore],
  );
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Import NI Counter Read Task(s)"
      icon={<Icon.Logo.NI />}
      onSelect={handleSelect}
    />
  );
};
ImportCounterReadCommand.key = "ni-import-counter-read-task";
ImportCounterReadCommand.commandName = "Import NI Counter Read Task(s)";
ImportCounterReadCommand.sortOrder = -1;
ImportCounterReadCommand.useVisible = useVisible;

export const ImportDigitalReadCommand: Palette.Command = ({
  placeLayout,
  handleError,
  store,
  client,
  fluxStore,
  ...listProps
}) => {
  const handleSelect = useCallback(
    () => importDigitalRead({ placeLayout, handleError, store, client, fluxStore }),
    [placeLayout, handleError, store, client, fluxStore],
  );
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Import NI Digital Read Task(s)"
      icon={<Icon.Logo.NI />}
      onSelect={handleSelect}
    />
  );
};
ImportDigitalReadCommand.key = "ni-import-digital-read-task";
ImportDigitalReadCommand.commandName = "Import NI Digital Read Task(s)";
ImportDigitalReadCommand.sortOrder = -1;
ImportDigitalReadCommand.useVisible = useVisible;

export const ImportDigitalWriteCommand: Palette.Command = ({
  placeLayout,
  handleError,
  store,
  client,
  fluxStore,
  ...listProps
}) => {
  const handleSelect = useCallback(
    () => importDigitalWrite({ placeLayout, handleError, store, client, fluxStore }),
    [placeLayout, handleError, store, client, fluxStore],
  );
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Import NI Digital Write Task(s)"
      icon={<Icon.Import />}
      onSelect={handleSelect}
    />
  );
};
ImportDigitalWriteCommand.key = "ni-import-digital-write-task";
ImportDigitalWriteCommand.commandName = "Import NI Digital Write Task(s)";
ImportDigitalWriteCommand.sortOrder = -1;
ImportDigitalWriteCommand.useVisible = useVisible;

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
  ImportAnalogReadCommand,
  ImportAnalogWriteCommand,
  ImportCounterReadCommand,
  ImportDigitalReadCommand,
  ImportDigitalWriteCommand,
  ToggleScannerCommand,
];
