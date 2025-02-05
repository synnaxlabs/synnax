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
import { Table } from "@/table";
import { ImportIcon } from "@/table/services/Icon";
import { import_ } from "@/table/services/import";

const CREATE_COMMAND: Palette.Command = {
  key: "create-table",
  name: "Create Table",
  icon: <Icon.Table />,
  onSelect: ({ placeLayout }) => placeLayout(Table.create({})),
};

const IMPORT_COMMAND: Palette.Command = {
  key: "import-table",
  name: "Import Table(s)",
  icon: <ImportIcon />,
  onSelect: (ctx: Import.ImportArgs) => void import_(ctx),
};

export const COMMANDS = [CREATE_COMMAND, IMPORT_COMMAND];
