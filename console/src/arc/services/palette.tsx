// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { Arc } from "@/arc";
import { type Palette } from "@/palette";

const CREATE_COMMAND: Palette.Command = {
  key: "create_arc",
  name: "Create an Arc",
  icon: <Icon.Arc />,
  onSelect: ({ placeLayout }) => placeLayout(Arc.createEditor()),
};

const OPEN_EXPLORER_COMMAND: Palette.Command = {
  key: "open_arc_explorer",
  name: "Open Arc Explorer",
  icon: <Icon.Explore />,
  onSelect: ({ placeLayout }) => placeLayout(Arc.EXPLORER_LAYOUT),
};

export const COMMANDS = [CREATE_COMMAND, OPEN_EXPLORER_COMMAND];
