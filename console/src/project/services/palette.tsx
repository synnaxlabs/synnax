// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { project } from "@synnaxlabs/client";
import { Access, Project as PProject } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { useFileIngesters } from "@/import/FileIngestersProvider";
import { Palette } from "@/palette";
import { Project } from "@/project";
import { import_ } from "@/project/services/import";

const useCreateVisible = () => Access.useCreateGranted(project.TYPE_ONTOLOGY_ID);
const useViewVisible = () => Access.useRetrieveGranted(project.TYPE_ONTOLOGY_ID);

const CREATE_COMMAND_NAME = "Create a project";

const CreateCommand: Palette.Command = (listProps) => {
  const open = Project.useCreateModal();
  const handleSelect = useCallback(() => open(), [open]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name={CREATE_COMMAND_NAME}
      icon={<PProject.CreateIcon />}
      onSelect={handleSelect}
    />
  );
};
CreateCommand.key = "project_create";
CreateCommand.commandName = CREATE_COMMAND_NAME;
CreateCommand.useVisible = useCreateVisible;

const IMPORT_COMMAND_NAME = "Import a project";

const ImportProjectCommand: Palette.Command = ({
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
      name={IMPORT_COMMAND_NAME}
      icon={<PProject.ImportIcon />}
      onSelect={handleSelect}
    />
  );
};
ImportProjectCommand.key = "project_import";
ImportProjectCommand.commandName = IMPORT_COMMAND_NAME;
ImportProjectCommand.sortOrder = -1;
ImportProjectCommand.useVisible = useCreateVisible;

const EXPORT_COMMAND_NAME = "Export current project";

const ExportProjectCommand: Palette.Command = (listProps) => {
  const handleExport = Project.useExport();
  const handleSelect = useCallback(() => handleExport(null), [handleExport]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name={EXPORT_COMMAND_NAME}
      icon={<PProject.ExportIcon />}
      onSelect={handleSelect}
    />
  );
};
ExportProjectCommand.key = "project_export";
ExportProjectCommand.commandName = EXPORT_COMMAND_NAME;
ExportProjectCommand.sortOrder = -1;
ExportProjectCommand.useVisible = useViewVisible;

export const COMMANDS = [CreateCommand, ImportProjectCommand, ExportProjectCommand];
