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
import { ANALOG_WRITE_LAYOUT } from "@/hardware/ni/task/AnalogWrite";
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

const createAnalogWriteCommand: Palette.Command = {
  key: "ni-create-analog-write-task",
  name: "Create an NI Analog Write Task",
  icon: <Icon.Logo.NI />,
  onSelect: ({ placeLayout }) => placeLayout(ANALOG_WRITE_LAYOUT),
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
  onSelect: ({ client, addStatus, handleException }) => {
    if (client == null) throw new Error("Cannot reach server");
    client.hardware.tasks
      .retrieveByName<ScanConfig>("ni scanner")
      .then(({ payload, config }) =>
        client.hardware.tasks.create<ScanConfig>({
          ...payload,
          config: { ...config, enabled: !config.enabled },
        }),
      )
      .then(({ config: { enabled } }) =>
        addStatus({
          variant: "success",
          message: `NI device scanning ${enabled ? "disabled" : "enabled"}`,
        }),
      )
      .catch(handleException);
  },
};

export const COMMANDS = [
  createAnalogReadCommand,
  createAnalogWriteCommand,
  createDigitalWriteCommand,
  createDigitalReadCommand,
  toggleScanner,
];
