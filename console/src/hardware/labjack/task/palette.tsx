// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { createReadLayout } from "@/hardware/labjack/task/Read";
import { createWriteLayout } from "@/hardware/labjack/task/Write";
import { type Command } from "@/palette/Palette";

const createReadCommand: Command = {
  key: "labjack-create-read-task",
  name: "Create LabJack Read Task",
  icon: <Icon.Logo.LabJack />,
  onSelect: ({ placeLayout }) => placeLayout(() => createReadLayout({ create: true })),
};

const createWriteCommand: Command = {
  key: "labjack-create-write-task",
  name: "Create LabJack Write Task",
  icon: <Icon.Logo.LabJack />,
  onSelect: ({ placeLayout }) => placeLayout(createWriteLayout({ create: true })),
};

export const COMMANDS = [createReadCommand, createWriteCommand];
