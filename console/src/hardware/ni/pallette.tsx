// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { configureAnalogReadLayout } from "@/hardware/ni/task/AnalogRead";
import { configureDigitalReadLayout } from "@/hardware/ni/task/ConfigureDigitalRead";
import { configureDigitalWriteLayout } from "@/hardware/ni/task/ConfigureDigitalWrite";
import { Command } from "@/palette/Palette";

export const createAnalogReadTaskCommand: Command = {
  key: "ni-create-analog-read-task",
  name: "NI - Create a New Analog Read Task",
  icon: <Icon.Logo.NI />,
  onSelect: ({ placeLayout }) => placeLayout(configureAnalogReadLayout),
};

export const createDigitalWriteTaskCommand: Command = {
  key: "ni-create-digital-write-task",
  name: "NI - Create a New Digital Write Task",
  icon: <Icon.Logo.NI />,
  onSelect: ({ placeLayout }) => placeLayout(configureDigitalWriteLayout),
};

export const createDigitalReadTaskCommand: Command = {
  key: "ni-create-digital-read-task",
  name: "NI - Create a New Digital Read Task",
  icon: <Icon.Logo.NI />,
  onSelect: ({ placeLayout }) => placeLayout(configureDigitalReadLayout),
};

export const COMMANDS = [
  createAnalogReadTaskCommand,
  createDigitalWriteTaskCommand,
  createDigitalReadTaskCommand,
];
