// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { importPlot } from "@/lineplot/file";
import { create } from "@/lineplot/LinePlot";
import { ImportIcon } from "@/lineplot/services/Icon";
import { type Command } from "@/palette/Palette";
import { Workspace } from "@/workspace";

export const createLinePlotCommand: Command = {
  key: "create-line-plot",
  name: "Create a Line Plot",
  icon: <Icon.Visualize />,
  onSelect: ({ placeLayout }) => placeLayout(create({})),
};

export const importLinePlotCommand: Command = {
  key: "import-line-plot",
  name: "Import Line Plot",
  icon: <ImportIcon />,
  onSelect: ({ placeLayout, ...props }) => {
    const { store } = props;
    const state = store.getState();
    const activeWorkspaceKey = Workspace.selectActiveKey(state);
    importPlot({
      activeWorkspaceKey,
      place: placeLayout,
      dispatch: store.dispatch,
      ...props,
    });
  },
};

export const COMMANDS = [createLinePlotCommand, importLinePlotCommand];
