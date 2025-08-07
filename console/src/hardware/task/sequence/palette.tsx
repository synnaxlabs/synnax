// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { import_ } from "@/hardware/task/sequence/import";
import { createLayout } from "@/hardware/task/sequence/Sequence";
import { type Palette } from "@/palette";

const CREATE_COMMAND: Palette.Command = {
  key: "create-control-sequence",
  name: "Create a Control Sequence",
  icon: <Icon.Control />,
  onSelect: ({ placeLayout, rename, handleError }) => {
    handleError(async () => {
      const layout = await createLayout({ rename });
      if (layout != null) placeLayout(layout);
    }, "Failed to create a control sequence");
  },
};

const IMPORT_COMMAND: Palette.Command = {
  key: "import-control-sequence",
  name: "Import a Control Sequence",
  icon: <Icon.Control />,
  sortOrder: -1,
  onSelect: import_,
};

export const COMMANDS = [CREATE_COMMAND, IMPORT_COMMAND];
