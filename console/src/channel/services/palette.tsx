// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { Channel } from "@/channel";
import { Beta } from "@/components";
import { type Palette } from "@/palette";

const CREATE_COMMAND: Palette.Command = {
  icon: <Icon.Channel />,
  name: "Create a Channel",
  key: "create-channel",
  onSelect: ({ placeLayout }) => placeLayout(Channel.CREATE_LAYOUT),
};

const CREATE_CALCULATED_COMMAND: Palette.Command = {
  icon: <Icon.Channel />,
  name: "Create a Calculated Channel",
  key: "create-calculated-channel",
  onSelect: ({ placeLayout }) => placeLayout(Channel.CALCULATED_LAYOUT),
  endContent: [<Beta.Tag key="beta-tag" />],
};

export const COMMANDS = [CREATE_COMMAND, CREATE_CALCULATED_COMMAND];
