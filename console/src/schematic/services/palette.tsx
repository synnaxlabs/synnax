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
import { importSchematic } from "@/schematic/file";
import { Workspace } from "@/workspace";
import { ImportIcon } from "@/schematic/services/Icon";

export const createCommand: Command = {
  key: "create-schematic",
  name: "Create a Schematic",
  icon: <Icon.Schematic />,
  onSelect: ({ placeLayout }) => placeLayout(create({})),
  visible: (state) => selectHasPermission(state),
};

export const importSchematicCommand: Command = {
  key: "import-schematic",
  name: "Import Schematic",
  icon: <ImportIcon />,
  onSelect: ({ placeLayout, ...props }) => {
    const { store } = props;
    const state = store.getState();
    const activeWorkspaceKey = Workspace.selectActiveKey(state);
    importSchematic({
      activeWorkspaceKey,
      placer: placeLayout,
      dispatch: store.dispatch,
      ...props,
    });
  },
};

export const COMMANDS = [createCommand, importSchematicCommand];
