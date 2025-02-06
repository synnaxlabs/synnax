// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Icon as PIcon } from "@synnaxlabs/pluto";

import { Task } from "@/hardware/labjack/task";
import { type Palette } from "@/palette";

const CREATE_READ_TASK_COMMAND: Palette.Command = {
  key: "labjack-create-read-task",
  name: "Create a LabJack Read Task",
  icon: (
    <PIcon.Create>
      <Icon.Logo.LabJack />
    </PIcon.Create>
  ),
  onSelect: ({ placeLayout }) => placeLayout(Task.createReadLayout({ create: true })),
};

const CREATE_WRITE_TASK_COMMAND: Palette.Command = {
  key: "labjack-create-write-task",
  name: "Create a LabJack Write Task",
  icon: (
    <PIcon.Create>
      <Icon.Logo.LabJack />
    </PIcon.Create>
  ),
  onSelect: ({ placeLayout }) => placeLayout(Task.createWriteLayout({ create: true })),
};

export const COMMANDS = [CREATE_READ_TASK_COMMAND, CREATE_WRITE_TASK_COMMAND];
