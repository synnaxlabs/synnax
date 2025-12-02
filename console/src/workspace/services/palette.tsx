// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { workspace } from "@synnaxlabs/client";
import { Access, Icon } from "@synnaxlabs/pluto";

import { type Palette } from "@/palette";
import { Workspace } from "@/workspace";
import { ImportIcon } from "@/workspace/services/Icon";
import { import_ } from "@/workspace/services/import";

const CREATE_COMMAND: Palette.Command = {
  key: "workspace-create",
  name: "Create a Workspace",
  icon: <Icon.Workspace />,
  onSelect: ({ placeLayout }) => placeLayout(Workspace.CREATE_LAYOUT),
  visible: ({ store, client }) =>
    Access.editGranted({ id: workspace.ontologyID(""), store, client }),
};

const IMPORT_COMMAND: Palette.Command = {
  key: "workspace-import",
  name: "Import a Workspace",
  sortOrder: -1,
  icon: <ImportIcon />,
  onSelect: import_,
  visible: ({ store, client }) =>
    Access.editGranted({ id: workspace.ontologyID(""), store, client }),
};

const EXPORT_COMMAND: Palette.Command = {
  key: "workspace-export",
  name: "Export Current Workspace",
  sortOrder: -1,
  icon: <Icon.Workspace />,
  onSelect: (ctx) => Workspace.export_(null, ctx),
  visible: ({ store, client }) =>
    Access.viewGranted({ id: workspace.ontologyID(""), store, client }),
};

export const COMMANDS = [CREATE_COMMAND, IMPORT_COMMAND, EXPORT_COMMAND];
