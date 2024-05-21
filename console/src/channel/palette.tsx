// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createLayout } from "@/channel/Create";
import { Command, CommandSelectionContext } from "@/palette/Palette";
import { Icon } from "@synnaxlabs/media";

export const createChannelCommand: Command = {
  icon: <Icon.Channel />,
  name: "Create Channel",
  key: "create-channel",
  onSelect: (ctx: CommandSelectionContext) => {
    ctx.placeLayout(createLayout);
  },
};

export const COMMANDS = [createChannelCommand];
