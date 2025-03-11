// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { NULL_CLIENT_ERROR } from "@/errors";
import { ANALOG_READ_LAYOUT } from "@/hardware/ni/task/AnalogRead";
import { ANALOG_WRITE_LAYOUT } from "@/hardware/ni/task/AnalogWrite";
import { DIGITAL_READ_LAYOUT } from "@/hardware/ni/task/DigitalRead";
import { DIGITAL_WRITE_LAYOUT } from "@/hardware/ni/task/DigitalWrite";
import { SCAN_NAME, type ScanConfig } from "@/hardware/ni/task/types";
import { type Palette } from "@/palette";

const CREATE_ANALOG_READ_COMMAND: Palette.Command = {
  key: "ni-create-analog-read-task",
  name: "Create an NI Analog Read Task",
  icon: <Icon.Logo.NI />,
  onSelect: ({ placeLayout }) => placeLayout(ANALOG_READ_LAYOUT),
};

const CREATE_ANALOG_WRITE_COMMAND: Palette.Command = {
  key: "ni-create-analog-write-task",
  name: "Create an NI Analog Write Task",
  icon: <Icon.Logo.NI />,
  onSelect: ({ placeLayout }) => placeLayout(ANALOG_WRITE_LAYOUT),
};

const CREATE_DIGITAL_WRITE_COMMAND: Palette.Command = {
  key: "ni-create-digital-write-task",
  name: "Create an NI Digital Write Task",
  icon: <Icon.Logo.NI />,
  onSelect: ({ placeLayout }) => placeLayout(DIGITAL_WRITE_LAYOUT),
};

const CREATE_DIGITAL_READ_COMMAND: Palette.Command = {
  key: "ni-create-digital-read-task",
  name: "Create an NI Digital Read Task",
  icon: <Icon.Logo.NI />,
  onSelect: ({ placeLayout }) => placeLayout(DIGITAL_READ_LAYOUT),
};

const TOGGLE_SCAN_TASK_COMMAND: Palette.Command = {
  key: "ni-toggle-scan-task",
  name: "Toggle NI Device Scanner",
  icon: <Icon.Logo.NI />,
  onSelect: ({ client, addStatus, handleException }) => {
    handleException(async () => {
      if (client == null) throw NULL_CLIENT_ERROR;
      const { payload, config } =
        await client.hardware.tasks.retrieveByName<ScanConfig>(SCAN_NAME);
      const {
        config: { enabled },
      } = await client.hardware.tasks.create<ScanConfig>({
        ...payload,
        config: { ...config, enabled: !config.enabled },
      });
      addStatus({
        variant: "success",
        message: `NI device scanning ${enabled ? "enabled" : "disabled"}`,
      });
    }, "Failed to toggle NI device scanner");
  },
};

export const COMMANDS = [
  CREATE_ANALOG_READ_COMMAND,
  CREATE_ANALOG_WRITE_COMMAND,
  CREATE_DIGITAL_WRITE_COMMAND,
  CREATE_DIGITAL_READ_COMMAND,
  TOGGLE_SCAN_TASK_COMMAND,
];
