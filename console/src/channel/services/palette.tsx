// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { createLayout } from "@/channel/Create";
import { createCalculatedLayout } from "@/channel/CreateCalculated";
import { type Command, type CommandSelectionContext } from "@/palette/Palette";

export const createChannelCommand: Command = {
  icon: <Icon.Channel />,
  name: "Create Channel",
  key: "create-channel",
  onSelect: (ctx: CommandSelectionContext) => {
    ctx.placeLayout(createLayout);
  },
};

export const createCalculatedChannelCommand: Command = {
  icon: <Icon.Channel />,
  name: "Create Calculated Channel",
  key: "create-calculated-channel",
  onSelect: (ctx: CommandSelectionContext) => {
    ctx.placeLayout(createCalculatedLayout);
  },
};

export const COMMANDS = [createChannelCommand, createCalculatedChannelCommand];
