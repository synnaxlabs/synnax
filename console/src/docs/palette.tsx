// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { LAYOUT } from "@/docs/Docs";
import { type Palette } from "@/palette";

const READ_COMMAND: Palette.Command = {
  key: "read-the-docs",
  name: "Read the documentation",
  icon: <Icon.QuestionMark />,
  onSelect: ({ placeLayout }) => placeLayout(LAYOUT),
};

export const COMMANDS = [READ_COMMAND];
