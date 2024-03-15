// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { manageWindowLayout } from "@/label/Manage";
import { type Command } from "@/palette/Palette";

export const manageCommand: Command = {
  key: "manage-labels",
  name: "Manage labels",
  icon: <Icon.Annotate />,
  onSelect: ({ placeLayout: layoutPlacer }) => layoutPlacer(manageWindowLayout),
};

export const COMMANDS = [manageCommand];
