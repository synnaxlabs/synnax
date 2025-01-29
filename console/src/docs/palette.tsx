// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { createLayout } from "@/docs/Docs";
import { type Palette } from "@/palette";

const READ_COMMAND: Palette.Command = {
  key: "read-the-docs",
  name: "Read the docs",
  icon: <Icon.QuestionMark />,
  onSelect: ({ placeLayout }) => placeLayout(createLayout()),
};

export const COMMANDS = [READ_COMMAND];
