// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { ANALOG_READ_LAYOUT } from "@/hardware/ni/task/AnalogRead";
import { DIGITAL_READ_LAYOUT } from "@/hardware/ni/task/DigitalRead";
import { DIGITAL_WRITE_LAYOUT } from "@/hardware/ni/task/DigitalWrite";
import { type ScanConfig } from "@/hardware/ni/task/types";
import { type Palette } from "@/palette";

const createAnalogReadCommand: Palette.Command = {
  key: "ni-create-analog-read-task",
  name: "Create an NI Analog Read Task",
  icon: <Icon.Logo.NI />,
  onSelect: ({ placeLayout }) => placeLayout(ANALOG_READ_LAYOUT),
};

const createDigitalWriteCommand: Palette.Command = {
  key: "ni-create-digital-write-task",
  name: "Create an NI Digital Write Task",
  icon: <Icon.Logo.NI />,
  onSelect: ({ placeLayout }) => placeLayout(DIGITAL_WRITE_LAYOUT),
};

const createDigitalReadCommand: Palette.Command = {
  key: "ni-create-digital-read-task",
  name: "Create an NI Digital Read Task",
  icon: <Icon.Logo.NI />,
  onSelect: ({ placeLayout }) => placeLayout(DIGITAL_READ_LAYOUT),
};

const toggleScanner: Palette.Command = {
  key: "toggle-ni-scan-task",
  name: "Toggle NI Device Scanner",
  icon: <Icon.Logo.NI />,
  onSelect: async ({ client, addStatus, handleException }) => {
    try {
      if (client == null) throw new Error("Cannot reach server");
      const tsk = await client.hardware.tasks.retrieveByName<ScanConfig>("ni scanner");
      const enabled = tsk.config.enabled ?? true;
      await client.hardware.tasks.create<ScanConfig>({
        ...tsk.payload,
        config: { ...tsk.config, enabled: !enabled },
      });
      addStatus({
        variant: "success",
        message: `NI device scanning ${enabled ? "disabled" : "enabled"}`,
      });
    } catch (e) {
      handleException(e, "Failed to toggle NI scan task");
    }
  },
};

export const COMMANDS = [
  createAnalogReadCommand,
  createDigitalWriteCommand,
  createDigitalReadCommand,
  toggleScanner,
];
