// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Palette } from "@/palette";
import { Table } from "@/table";
import { CreateIcon, ImportIcon } from "@/table/services/Icon";
import { import_ } from "@/table/services/import";

const useVisible = () => Access.useUpdateGranted(table.TYPE_ONTOLOGY_ID);

export const CreateCommand: Palette.Command = ({ placeLayout, ...listProps }) => {
  const handleSelect = useCallback(() => placeLayout(Table.create()), [placeLayout]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Create a table"
      icon={<CreateIcon />}
      onSelect={handleSelect}
    />
  );
};
CreateCommand.key = "create-table";
CreateCommand.commandName = "Create a table";
CreateCommand.useVisible = useVisible;

export const ImportCommand: Palette.Command = ({
  placeLayout,
  handleError,
  store,
  client,
  fluxStore,
  ...listProps
}) => {
  const handleSelect = useCallback(
    () => import_({ placeLayout, handleError, store, client, fluxStore }),
    [placeLayout, handleError, store, client, fluxStore],
  );
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Import Table(s)"
      icon={<ImportIcon />}
      onSelect={handleSelect}
    />
  );
};
ImportCommand.key = "import-table";
ImportCommand.commandName = "Import Table(s)";
ImportCommand.sortOrder = -1;
ImportCommand.useVisible = useVisible;

export const COMMANDS = [CreateCommand, ImportCommand];
