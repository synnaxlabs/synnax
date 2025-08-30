// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Palette } from "@/palette";
import { Schematic } from "@/schematic";
import { CreateIcon, ImportIcon } from "@/schematic/services/Icon";
import { import_ } from "@/schematic/services/import";
import { createCreateLayout } from "@/schematic/symbols/edit/Edit";

const CREATE_COMMAND: Palette.Command = {
  key: "create-schematic",
  name: "Create a Schematic",
  icon: <CreateIcon />,
  onSelect: ({ placeLayout }) => placeLayout(Schematic.create()),
  visible: Schematic.selectHasPermission,
};

const IMPORT_COMMAND: Palette.Command = {
  key: "import-schematic",
  name: "Import Schematic(s)",
  sortOrder: -1,
  icon: <ImportIcon />,
  onSelect: import_,
  visible: Schematic.selectHasPermission,
};

const CREATE_SYMBOL_COMMAND: Palette.Command = {
  key: "create-symbol",
  name: "Create Symbol",
  icon: <CreateIcon />,
  onSelect: ({ placeLayout }) => placeLayout(createCreateLayout()),
  visible: Schematic.selectHasPermission,
};

export const COMMANDS = [CREATE_COMMAND, IMPORT_COMMAND, CREATE_SYMBOL_COMMAND];
