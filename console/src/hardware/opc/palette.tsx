// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { createConfigureLayout } from "@/hardware/opc/device/Configure";
import { configureReadLayout } from "@/hardware/opc/task/ReadTask";
import { type Command } from "@/palette/Palette";

export const connectServerCommand: Command = {
  key: "opc-connect-server",
  name: "Connect an OPC UA Server",
  icon: <Icon.Logo.OPC />,
  onSelect: ({ placeLayout }) => placeLayout(createConfigureLayout()),
};

export const createReadTaskCommand: Command = {
  key: "opc-create-read-task",
  name: "Create an OPC UA Read Task",
  icon: <Icon.Logo.OPC />,
  onSelect: ({ placeLayout }) => placeLayout(configureReadLayout(true)),
};

export const COMMANDS = [connectServerCommand, createReadTaskCommand];
