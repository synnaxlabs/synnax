// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { workspace } from "@synnaxlabs/client";
import { Access, Workspace as PWorkspace } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { useFileIngesters } from "@/import/FileIngestersProvider";
import { Palette } from "@/palette";
import { Workspace } from "@/workspace";
import { import_ } from "@/workspace/services/import";

const useCreateVisible = () => Access.useCreateGranted(workspace.TYPE_ONTOLOGY_ID);
const useViewVisible = () => Access.useRetrieveGranted(workspace.TYPE_ONTOLOGY_ID);

const CreateCommand = Palette.createSimpleCommand({
  key: "workspace-create",
  name: "Create a workspace",
  icon: <PWorkspace.CreateIcon />,
  layout: Workspace.CREATE_LAYOUT,
  useVisible: useCreateVisible,
});

const ImportWorkspaceCommand: Palette.Command = ({
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
      name="Import a workspace"
      icon={<PWorkspace.ImportIcon />}
      onSelect={handleSelect}
    />
  );
};
ImportWorkspaceCommand.key = "workspace-import";
ImportWorkspaceCommand.commandName = "Import a workspace";
ImportWorkspaceCommand.sortOrder = -1;
ImportWorkspaceCommand.useVisible = useCreateVisible;

const ExportWorkspaceCommand: Palette.Command = (listProps) => {
  const handleExport = Workspace.useExport();
  const handleSelect = useCallback(() => handleExport(null), [handleExport]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Export current workspace"
      icon={<PWorkspace.ExportIcon />}
      onSelect={handleSelect}
    />
  );
};
ExportWorkspaceCommand.key = "workspace-export";
ExportWorkspaceCommand.commandName = "Export current workspace";
ExportWorkspaceCommand.sortOrder = -1;
ExportWorkspaceCommand.useVisible = useViewVisible;

export const COMMANDS = [CreateCommand, ImportWorkspaceCommand, ExportWorkspaceCommand];
