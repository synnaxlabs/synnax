// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Command } from "@/palette/Palette";
import { analogReadTaskLayout } from "@/hardware/ni/AnalogReadTask";

export const createAnalogReadTaskCommand: Command = {
  key: "ni-create-analog-read-task",
  name: "NI - Create a New Analog Read Task",
  icon: <Icon.PID />,
  onSelect: ({ placeLayout }) => placeLayout(analogReadTaskLayout),
};

export const COMMANDS = [createAnalogReadTaskCommand];
