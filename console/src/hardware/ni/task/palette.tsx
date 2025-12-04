// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, UnexpectedError } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/pluto";

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

const CREATE_COUNTER_READ_COMMAND: Palette.Command = {
  key: "ni-create-counter-read-task",
  name: "Create an NI Counter Read Task",
  icon: <Icon.Logo.NI />,
  onSelect: ({ placeLayout }) => placeLayout(COUNTER_READ_LAYOUT),
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

const IMPORT_ANALOG_READ_COMMAND: Palette.Command = {
  key: "ni-import-analog-read-task",
  name: "Import NI Analog Read Task(s)",
  sortOrder: -1,
  icon: <Icon.Logo.NI />,
  onSelect: importAnalogRead,
};

const IMPORT_ANALOG_WRITE_COMMAND: Palette.Command = {
  key: "ni-import-analog-write-task",
  name: "Import NI Analog Write Task(s)",
  sortOrder: -1,
  icon: <Icon.Logo.NI />,
  onSelect: importAnalogWrite,
};

const IMPORT_COUNTER_READ_COMMAND: Palette.Command = {
  key: "ni-import-counter-read-task",
  name: "Import NI Counter Read Task(s)",
  sortOrder: -1,
  icon: <Icon.Logo.NI />,
  onSelect: importCounterRead,
};

const IMPORT_DIGITAL_READ_COMMAND: Palette.Command = {
  key: "ni-import-digital-read-task",
  name: "Import NI Digital Read Task(s)",
  sortOrder: -1,
  icon: <Icon.Logo.NI />,
  onSelect: importDigitalRead,
};

const IMPORT_DIGITAL_WRITE_COMMAND: Palette.Command = {
  key: "ni-import-digital-write-task",
  name: "Import NI Digital Write Task(s)",
  sortOrder: -1,
  icon: <Icon.Import />,
  onSelect: importDigitalWrite,
};

const TOGGLE_SCAN_TASK_COMMAND: Palette.Command = {
  key: "ni-toggle-scan-task",
  name: "Toggle the NI Device Scanner",
  icon: <Icon.Logo.NI />,
  onSelect: ({ client, addStatus, handleError }) => {
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
    }, "Failed to toggle NI device scanner");
  },
};

export const COMMANDS = [
  CREATE_ANALOG_READ_COMMAND,
  CREATE_ANALOG_WRITE_COMMAND,
  CREATE_COUNTER_READ_COMMAND,
  CREATE_DIGITAL_WRITE_COMMAND,
  CREATE_DIGITAL_READ_COMMAND,
  IMPORT_ANALOG_READ_COMMAND,
  IMPORT_ANALOG_WRITE_COMMAND,
  IMPORT_COUNTER_READ_COMMAND,
  IMPORT_DIGITAL_READ_COMMAND,
  IMPORT_DIGITAL_WRITE_COMMAND,
  TOGGLE_SCAN_TASK_COMMAND,
];
