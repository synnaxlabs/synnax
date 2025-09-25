// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Ranger } from "@synnaxlabs/pluto";

import { type Palette } from "@/palette";
import { Range } from "@/range";

const CREATE_COMMAND: Palette.Command = {
  key: "define-range",
  name: "Create a Range",
  icon: <Ranger.CreateIcon />,
  onSelect: ({ placeLayout }) => placeLayout(Range.CREATE_LAYOUT),
};

const OPEN_EXPLORER_COMMAND: Palette.Command = {
  key: "open-explorer",
  name: "Open Range Explorer",
  icon: <Icon.Explore />,
  onSelect: ({ placeLayout }) => placeLayout(Range.EXPLORER_LAYOUT),
};

export const COMMANDS = [CREATE_COMMAND, OPEN_EXPLORER_COMMAND];
