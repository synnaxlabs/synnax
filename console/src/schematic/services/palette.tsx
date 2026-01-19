// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Palette } from "@/palette";
import { Schematic } from "@/schematic";
import { CreateIcon, ImportIcon } from "@/schematic/services/Icon";
import { import_ } from "@/schematic/services/import";

const useVisible = () => Access.useUpdateGranted(schematic.TYPE_ONTOLOGY_ID);

export const CreateCommand: Palette.Command = ({ placeLayout, ...listProps }) => {
  const handleSelect = useCallback(
    () => placeLayout(Schematic.create()),
    [placeLayout],
  );
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Create a Schematic"
      icon={<CreateIcon />}
      onSelect={handleSelect}
    />
  );
};
CreateCommand.key = "create-schematic";
CreateCommand.commandName = "Create a Schematic";
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
      name="Import Schematic(s)"
      icon={<ImportIcon />}
      onSelect={handleSelect}
    />
  );
};
ImportCommand.key = "import-schematic";
ImportCommand.commandName = "Import Schematic(s)";
ImportCommand.sortOrder = -1;
ImportCommand.useVisible = useVisible;

export const COMMANDS = [CreateCommand, ImportCommand];
