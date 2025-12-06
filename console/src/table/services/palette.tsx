// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";

import { type Palette } from "@/palette";
import { Table } from "@/table";
import { CreateIcon, ImportIcon } from "@/table/services/Icon";
import { import_ } from "@/table/services/import";

const CREATE_COMMAND: Palette.Command = {
  key: "create-table",
  name: "Create a Table",
  icon: <CreateIcon />,
  onSelect: ({ placeLayout }) => placeLayout(Table.create()),
  visible: ({ store, client }) =>
    Access.editGranted({ id: table.TYPE_ONTOLOGY_ID, store, client }),
};

const IMPORT_COMMAND: Palette.Command = {
  key: "import-table",
  name: "Import Table(s)",
  sortOrder: -1,
  icon: <ImportIcon />,
  onSelect: import_,
  visible: ({ store, client }) =>
    Access.editGranted({ id: table.TYPE_ONTOLOGY_ID, store, client }),
};

export const COMMANDS = [CREATE_COMMAND, IMPORT_COMMAND];
