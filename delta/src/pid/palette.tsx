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
import { create } from "@/pid/slice";

export const createCommand: Command = {
  key: "create-pid",
  name: "Create a PID",
  icon: <Icon.Control />,
  onSelect: ({ placeLayout: layoutPlacer }) => layoutPlacer(create({})),
};

export const COMMANDS = [createCommand];
