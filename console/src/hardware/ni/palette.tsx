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

import { createAnalogReadLayout } from "@/hardware/ni/task/AnalogRead";
import { createDigitalReadLayout } from "@/hardware/ni/task/DigitalRead";
import { createDigitalWriteLayout } from "@/hardware/ni/task/DigitalWrite";
import { type ScanConfig } from "@/hardware/ni/task/types";
import { type Command } from "@/palette/Palette";

export const createAnalogReadTaskCommand: Command = {
  key: "ni-create-analog-read-task",
  name: "Create an NI Analog Read Task",
  icon: (
    <PIcon.Create>
      <Icon.Logo.NI />
    </PIcon.Create>
  ),
  onSelect: ({ placeLayout }) =>
    placeLayout(() => createAnalogReadLayout({ create: true })),
};

export const createDigitalWriteTaskCommand: Command = {
  key: "ni-create-digital-write-task",
  name: "Create an NI Digital Write Task",
  icon: (
    <PIcon.Create>
      <Icon.Logo.NI />
    </PIcon.Create>
  ),
  onSelect: ({ placeLayout }) =>
    placeLayout(createDigitalWriteLayout({ create: true })),
};

export const createDigitalReadTaskCommand: Command = {
  key: "ni-create-digital-read-task",
  name: "Create an NI Digital Read Task",
  icon: (
    <PIcon.Create>
      <Icon.Logo.NI />
    </PIcon.Create>
  ),
  onSelect: ({ placeLayout }) => placeLayout(createDigitalReadLayout({ create: true })),
};

export const toggleNIScanner: Command = {
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
          await client.hardware.tasks.retrieveByName<ScanConfig>("ni scanner");
        const enabled = tsk.config.enabled ?? true;
        client.hardware.tasks.create<ScanConfig>({
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
  createAnalogReadTaskCommand,
  createDigitalWriteTaskCommand,
  createDigitalReadTaskCommand,
  toggleNIScanner,
];
