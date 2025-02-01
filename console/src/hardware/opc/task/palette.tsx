// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { configureReadLayout } from "@/hardware/opc/task/Read";
import { createWriteLayout } from "@/hardware/opc/task/Write";
import { type Palette } from "@/palette";

const CREATE_READ_TASK_COMMAND: Palette.Command = {
  key: "opc-create-read-task",
  name: "Create an OPC UA Read Task",
  icon: <Icon.Logo.OPC />,
  onSelect: ({ placeLayout }) => placeLayout(configureReadLayout({ create: true })),
};

const CREATE_WRITE_TASK_COMMAND: Palette.Command = {
  key: "opc-create-write-task",
  name: "Create an OPC UA Write Task",
  icon: <Icon.Logo.OPC />,
  onSelect: ({ placeLayout }) => placeLayout(createWriteLayout({ create: true })),
};

export const COMMANDS = [CREATE_READ_TASK_COMMAND, CREATE_WRITE_TASK_COMMAND];
