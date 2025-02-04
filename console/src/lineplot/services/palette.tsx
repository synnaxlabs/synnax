// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { type Import } from "@/import";
import { LinePlot } from "@/lineplot";
import { ImportIcon } from "@/lineplot/services/Icon";
import { import_ } from "@/lineplot/services/import";
import { type Palette } from "@/palette";

const CREATE_COMMAND: Palette.Command = {
  key: "create-line-plot",
  name: "Create Line Plot",
  icon: <Icon.Visualize />,
  onSelect: ({ placeLayout }) => placeLayout(LinePlot.create({})),
};

const IMPORT_COMMAND: Palette.Command = {
  key: "import-line-plot",
  name: "Import Line Plot(s)",
  icon: <ImportIcon />,
  onSelect: (ctx: Import.ImportArgs) => void import_(ctx),
};

export const COMMANDS = [CREATE_COMMAND, IMPORT_COMMAND];
