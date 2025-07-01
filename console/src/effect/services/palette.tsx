// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { createEditLayout } from "@/effect/edit/layout";
import { type Palette } from "@/palette";

const CREATE_COMMAND: Palette.Command = {
  key: "create_effect",
  name: "Create an Effect",
  icon: <Icon.Effect />,
  onSelect: ({ placeLayout }) => placeLayout(createEditLayout()),
};

export const COMMANDS = [CREATE_COMMAND];
