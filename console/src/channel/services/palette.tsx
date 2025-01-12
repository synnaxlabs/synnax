// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { CREATE_LAYOUT } from "@/channel/Create";
import { createCalculatedLayout } from "@/channel/CreateCalculated";
import { type Command, type CommandSelectionContext } from "@/palette/Palette";
import { Version } from "@/version";

const CREATE_CMD: Command = {
  icon: <Icon.Channel />,
  name: "Create Channel",
  key: "create-channel",
  onSelect: ({ placeLayout }: CommandSelectionContext) => {
    placeLayout(CREATE_LAYOUT);
  },
};

const CREATE_CALCULATED_CMD: Command = {
  icon: <Icon.Channel />,
  name: "Create Calculated Channel",
  key: "create-calculated-channel",
  onSelect: ({ placeLayout }: CommandSelectionContext) => {
    placeLayout(createCalculatedLayout);
  },
  actions: [<Version.BetaTag key="beta-tag" />],
};

export const COMMANDS = [CREATE_CMD, CREATE_CALCULATED_CMD];
