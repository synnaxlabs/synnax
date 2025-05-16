// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { TOOLS_LAYOUT } from "@/debug/Tools";
import { type Palette } from "@/palette";

export const OPEN_TOOLS_COMMAND: Palette.Command = {
  key: "open-debug-tools",
  name: "Open Debug Tools",
  icon: <Icon.Channel />,
  onSelect: ({ placeLayout }) => placeLayout(TOOLS_LAYOUT),
};

export const COMMANDS = [OPEN_TOOLS_COMMAND];
