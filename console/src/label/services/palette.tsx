// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { Label } from "@/label";
import { type Palette } from "@/palette";

const EDIT_COMMAND: Palette.Command = {
  key: "edit-labels",
  name: "Edit Labels",
  icon: <Icon.Label />,
  onSelect: ({ placeLayout }) => placeLayout(Label.createEditLayout()),
};

export const COMMANDS = [EDIT_COMMAND];
