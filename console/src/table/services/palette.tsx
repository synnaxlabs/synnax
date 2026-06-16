// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { Access, Table as PTable } from "@synnaxlabs/pluto";

import { Palette } from "@/palette";
import { useCreate } from "@/table/useCreate";

const CreateCommand: Palette.Command = (listProps) => {
  const create = useCreate({});
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Create a table"
      icon={<PTable.CreateIcon />}
      onSelect={create}
    />
  );
};
CreateCommand.key = "create-table";
CreateCommand.commandName = "Create a table";
CreateCommand.useVisible = () => Access.useCreateGranted(table.TYPE_ONTOLOGY_ID);

export const COMMANDS = [CreateCommand];
