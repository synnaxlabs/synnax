// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { type Command } from "@/palette/Palette";
import { createLayout } from "@/range/CreateLayout";

export const defineCommand: Command = {
  key: "define-range",
  name: "Create a Range",
  icon: <Icon.Range />,
  onSelect: ({ placeLayout }) => placeLayout(createLayout({})),
};

export const COMMANDS = [defineCommand];
