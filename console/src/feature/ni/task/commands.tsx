// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, task, UnexpectedError } from "@synnaxlabs/client";
import { Access, Icon, Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { ANALOG_READ_LAYOUT } from "@/feature/ni/task/AnalogRead";
import { ANALOG_WRITE_LAYOUT } from "@/feature/ni/task/AnalogWrite";
import { COUNTER_READ_LAYOUT } from "@/feature/ni/task/CounterRead";
import { DIGITAL_READ_LAYOUT } from "@/feature/ni/task/DigitalRead";
import { DIGITAL_WRITE_LAYOUT } from "@/feature/ni/task/DigitalWrite";
import { SCAN_SCHEMAS, SCAN_TYPE } from "@/feature/ni/task/types";
import { Command } from "@/platform/command";

const useVisible = () => Access.useCreateGranted(task.TYPE_ONTOLOGY_ID);

const CreateAnalogReadCommand = Command.create({
  key: "ni_create_analog_read_task",
  name: "Create an NI Analog Read Task",
  icon: <Icon.Logo.NI />,
  useOnSelect: Command.createPlacerUseOnSelect(ANALOG_READ_LAYOUT),
  useVisible,
});

const CreateAnalogWriteCommand = Command.create({
  key: "ni_create_analog_write_task",
  name: "Create an NI Analog Write Task",
  icon: <Icon.Logo.NI />,
  useOnSelect: Command.createPlacerUseOnSelect(ANALOG_WRITE_LAYOUT),
  useVisible,
});

const CreateCounterReadCommand = Command.create({
  key: "ni_create_counter_read_task",
  name: "Create an NI Counter Read Task",
  icon: <Icon.Logo.NI />,
  useOnSelect: Command.createPlacerUseOnSelect(COUNTER_READ_LAYOUT),
  useVisible,
});

const CreateDigitalWriteCommand = Command.create({
  key: "ni_create_digital_write_task",
  name: "Create an NI Digital Write Task",
  icon: <Icon.Logo.NI />,
  useOnSelect: Command.createPlacerUseOnSelect(DIGITAL_WRITE_LAYOUT),
  useVisible,
});

const CreateDigitalReadCommand = Command.create({
  key: "ni_create_digital_read_task",
  name: "Create an NI Digital Read Task",
  icon: <Icon.Logo.NI />,
  useOnSelect: Command.createPlacerUseOnSelect(DIGITAL_READ_LAYOUT),
  useVisible,
});

const useToggleScanner = () => {
  const client = Synnax.use();
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  return useCallback(
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
          { ...payload, config: { ...config, enabled: !config.enabled } },
          SCAN_SCHEMAS,
        );
        addStatus({
          variant: "success",
          message: `NI device scanning ${enabled ? "enabled" : "disabled"}`,
        });
      }, "Failed to toggle NI device scanner"),
    [client, addStatus, handleError],
  );
};

const ToggleScannerCommand = Command.create({
  key: "ni_toggle_scan_task",
  name: "Toggle the NI Device Scanner",
  icon: <Icon.Logo.NI />,
  useOnSelect: useToggleScanner,
  useVisible: () => Access.useUpdateGranted(task.TYPE_ONTOLOGY_ID),
});

export const COMMANDS = [
  CreateAnalogReadCommand,
  CreateAnalogWriteCommand,
  CreateCounterReadCommand,
  CreateDigitalWriteCommand,
  CreateDigitalReadCommand,
  ToggleScannerCommand,
];
