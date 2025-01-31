// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { type Palette } from "@/palette";
import { Workspace } from "@/workspace";
import { ImportIcon } from "@/workspace/services/Icon";
import { import_ } from "@/workspace/services/import";

const CREATE_COMMAND: Palette.Command = {
  key: "workspace-create",
  name: "Create Workspace",
  icon: <Icon.Workspace />,
  onSelect: ({ placeLayout }) => placeLayout(Workspace.CREATE_WINDOW_LAYOUT),
};

const IMPORT_COMMAND: Palette.Command = {
  key: "workspace-import",
  name: "Import Workspace",
  icon: <ImportIcon />,
  onSelect: import_,
};

export const COMMANDS = [CREATE_COMMAND, IMPORT_COMMAND];
