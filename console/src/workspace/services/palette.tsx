// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { workspace } from "@synnaxlabs/client";
import { Access, Icon } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { useFileIngesters } from "@/import/FileIngestersProvider";
import { Palette } from "@/palette";
import { Workspace } from "@/workspace";
import { ImportIcon } from "@/workspace/services/Icon";
import { import_ } from "@/workspace/services/import";

const useUpdateVisible = () => Access.useUpdateGranted(workspace.TYPE_ONTOLOGY_ID);
const useViewVisible = () => Access.useRetrieveGranted(workspace.TYPE_ONTOLOGY_ID);

export const CreateCommand = Palette.createSimpleCommand({
  key: "workspace-create",
  name: "Create a Workspace",
  icon: <Icon.Workspace />,
  layout: Workspace.CREATE_LAYOUT,
  useVisible: useUpdateVisible,
});

export const ImportWorkspaceCommand: Palette.Command = ({
  placeLayout,
  handleError,
  store,
  client,
  fluxStore,
  ...listProps
}) => {
  const fileIngesters = useFileIngesters();
  const handleSelect = useCallback(
    () =>
      import_({ placeLayout, handleError, store, client, fluxStore, fileIngesters }),
    [placeLayout, handleError, store, client, fluxStore, fileIngesters],
  );
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Import a Workspace"
      icon={<ImportIcon />}
      onSelect={handleSelect}
    />
  );
};
ImportWorkspaceCommand.key = "workspace-import";
ImportWorkspaceCommand.commandName = "Import a Workspace";
ImportWorkspaceCommand.sortOrder = -1;
ImportWorkspaceCommand.useVisible = useUpdateVisible;

export const ExportWorkspaceCommand: Palette.Command = (listProps) => {
  const handleExport = Workspace.useExport();
  const handleSelect = useCallback(() => handleExport(null), [handleExport]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Export Current Workspace"
      icon={<Icon.Workspace />}
      onSelect={handleSelect}
    />
  );
};
ExportWorkspaceCommand.key = "workspace-export";
ExportWorkspaceCommand.commandName = "Export Current Workspace";
ExportWorkspaceCommand.sortOrder = -1;
ExportWorkspaceCommand.useVisible = useViewVisible;

export const COMMANDS = [CreateCommand, ImportWorkspaceCommand, ExportWorkspaceCommand];
