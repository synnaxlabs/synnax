// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { create } from "@/lineplot/LinePlot";
import { ImportIcon } from "@/lineplot/services/Icon";
import { import_ } from "@/lineplot/services/import";
import { type Command } from "@/palette/Palette";

const CREATE_COMMAND: Command = {
  key: "create-line-plot",
  name: "Create Line Plot",
  icon: <Icon.Visualize />,
  onSelect: ({ placeLayout }) => placeLayout(create({})),
};

const IMPORT_COMMAND: Command = {
  key: "import-line-plot",
  name: "Import Line Plot(s)",
  icon: <ImportIcon />,
  onSelect: import_,
};

export const COMMANDS = [CREATE_COMMAND, IMPORT_COMMAND];
