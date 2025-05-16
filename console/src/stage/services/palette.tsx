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
import { type Palette } from "@/palette";
import { Stage } from "@/stage";
import { ImportIcon } from "@/stage/services/Icon";
import { import_ } from "@/stage/services/import";

const CREATE_COMMAND: Palette.Command = {
  key: "create-stage",
  name: "Create a Stage",
  icon: <Icon.Stage />,
  onSelect: ({ placeLayout }) => placeLayout(Stage.create()),
  visible: Stage.selectHasPermission,
};

const IMPORT_COMMAND: Palette.Command = {
  key: "import-stage",
  name: "Import Stage(s)",
  icon: <ImportIcon />,
  onSelect: (ctx: Import.ImportArgs) => void import_(ctx),
  visible: Stage.selectHasPermission,
};

export const COMMANDS = [CREATE_COMMAND, IMPORT_COMMAND];
