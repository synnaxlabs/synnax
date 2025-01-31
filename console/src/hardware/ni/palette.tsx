// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Icon as PIcon } from "@synnaxlabs/pluto";

import { Task } from "@/hardware/ni/task";
import { type Palette } from "@/palette";

const CREATE_ANALOG_READ_TASK_COMMAND: Palette.Command = {
  key: "ni-create-analog-read-task",
  name: "Create an NI Analog Read Task",
  icon: (
    <PIcon.Create>
      <Icon.Logo.NI />
    </PIcon.Create>
  ),
  onSelect: ({ placeLayout }) =>
    placeLayout(Task.createAnalogReadLayout({ create: true })),
};

const CREATE_DIGITAL_WRITE_TASK_COMMAND: Palette.Command = {
  key: "ni-create-digital-write-task",
  name: "Create an NI Digital Write Task",
  icon: (
    <PIcon.Create>
      <Icon.Logo.NI />
    </PIcon.Create>
  ),
  onSelect: ({ placeLayout }) =>
    placeLayout(Task.createDigitalWriteLayout({ create: true })),
};

const CREATE_DIGITAL_READ_TASK_COMMAND: Palette.Command = {
  key: "ni-create-digital-read-task",
  name: "Create an NI Digital Read Task",
  icon: (
    <PIcon.Create>
      <Icon.Logo.NI />
    </PIcon.Create>
  ),
  onSelect: ({ placeLayout }) =>
    placeLayout(Task.createDigitalReadLayout({ create: true })),
};

const TOGGLE_NI_SCAN_TASK_COMMAND: Palette.Command = {
  key: "toggle-ni-scan-task",
  name: "Toggle NI Device Scanner",
  icon: (
    <PIcon.Create>
      <Icon.Logo.NI />
    </PIcon.Create>
  ),
  onSelect: ({ client, addStatus, handleException }) => {
    if (client == null) return;
    void (async () => {
      try {
        const tsk =
          await client.hardware.tasks.retrieveByName<Task.ScanConfig>("ni scanner");
        const enabled = tsk.config.enabled ?? true;
        await client.hardware.tasks.create<Task.ScanConfig>({
          ...tsk.payload,
          config: { enabled: !enabled },
        });
        addStatus({
          variant: "success",
          message: `NI device scanning ${!enabled ? "enabled" : "disabled"}`,
        });
      } catch (e) {
        handleException(e, "Failed to toggle NI scan task");
      }
    })();
  },
};

export const COMMANDS = [
  CREATE_ANALOG_READ_TASK_COMMAND,
  CREATE_DIGITAL_WRITE_TASK_COMMAND,
  CREATE_DIGITAL_READ_TASK_COMMAND,
  TOGGLE_NI_SCAN_TASK_COMMAND,
];
