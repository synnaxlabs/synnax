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

import { Palette } from "@/palette";
import { Workspace } from "@/workspace";
import { ImportIcon } from "@/workspace/services/Icon";
import { import_ } from "@/workspace/services/import";

const useUpdateVisible = () => Access.useUpdateGranted(workspace.TYPE_ONTOLOGY_ID);
const useViewVisible = () => Access.useRetrieveGranted(workspace.TYPE_ONTOLOGY_ID);

export const CreateCommand = Palette.createSimpleCommand({
  key: "workspace-create",
  name: "Create a workspace",
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
  fileIngestors,
  ...listProps
}) => {
  const handleSelect = useCallback(
    () =>
      import_({ placeLayout, handleError, store, client, fluxStore, fileIngestors }),
    [placeLayout, handleError, store, client, fluxStore, fileIngestors],
  );
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Import a workspace"
      icon={<ImportIcon />}
      onSelect={handleSelect}
    />
  );
};
ImportWorkspaceCommand.key = "workspace-import";
ImportWorkspaceCommand.commandName = "Import a workspace";
ImportWorkspaceCommand.sortOrder = -1;
ImportWorkspaceCommand.useVisible = useUpdateVisible;

export const ExportWorkspaceCommand: Palette.Command = ({
  handleError,
  client,
  store,
  confirm,
  addStatus,
  extractors,
  ...listProps
}) => {
  const handleSelect = useCallback(
    () =>
      Workspace.export_(null, {
        handleError,
        client,
        store,
        confirm,
        addStatus,
        extractors,
      }),
    [handleError, client, store, confirm, addStatus, extractors],
  );
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Export current workspace"
      icon={<Icon.Workspace />}
      onSelect={handleSelect}
    />
  );
};
ExportWorkspaceCommand.key = "workspace-export";
ExportWorkspaceCommand.commandName = "Export current workspace";
ExportWorkspaceCommand.sortOrder = -1;
ExportWorkspaceCommand.useVisible = useViewVisible;

export const COMMANDS = [CreateCommand, ImportWorkspaceCommand, ExportWorkspaceCommand];
