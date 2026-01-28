// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { LinePlot } from "@/lineplot";
import { CreateIcon, ImportIcon } from "@/lineplot/services/Icon";
import { import_ } from "@/lineplot/services/import";
import { Palette } from "@/palette";

const useVisible = () => Access.useUpdateGranted(lineplot.TYPE_ONTOLOGY_ID);

export const CreateCommand: Palette.Command = ({ placeLayout, ...listProps }) => {
  const handleSelect = useCallback(() => placeLayout(LinePlot.create()), [placeLayout]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Create a Line Plot"
      icon={<CreateIcon />}
      onSelect={handleSelect}
    />
  );
};
CreateCommand.key = "create-line-plot";
CreateCommand.commandName = "Create a Line Plot";
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
      name="Import Line Plot(s)"
      icon={<ImportIcon />}
      onSelect={handleSelect}
    />
  );
};
ImportCommand.key = "import-line-plot";
ImportCommand.commandName = "Import Line Plot(s)";
ImportCommand.sortOrder = -1;
ImportCommand.useVisible = useVisible;

export const COMMANDS = [CreateCommand, ImportCommand];
