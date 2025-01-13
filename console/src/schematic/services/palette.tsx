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
import { create } from "@/schematic/Schematic";
import { selectHasPermission } from "@/schematic/selectors";
import { ImportIcon } from "@/schematic/services/Icon";
import { import_ } from "@/schematic/services/import";

const CREATE_COMMAND: Command = {
  key: "create-schematic",
  name: "Create Schematic",
  icon: <Icon.Schematic />,
  onSelect: ({ placeLayout }) => placeLayout(create({})),
  visible: (state) => selectHasPermission(state),
};

const IMPORT_COMMAND: Command = {
  key: "import-schematic",
  name: "Import Schematic(s)",
  icon: <ImportIcon />,
  onSelect: import_,
};

export const COMMANDS = [CREATE_COMMAND, IMPORT_COMMAND];
