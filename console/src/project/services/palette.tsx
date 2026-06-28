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

const CreateCommand: Palette.Command = (listProps) => {
  const open = Project.useOpenCreate();
  const handleSelect = useCallback(() => open(), [open]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Create a project"
      icon={<PProject.CreateIcon />}
      onSelect={handleSelect}
    />
  );
};
CreateCommand.key = "project-create";
CreateCommand.commandName = "Create a project";
CreateCommand.useVisible = useCreateVisible;

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
      name="Import a project"
      icon={<PProject.ImportIcon />}
      onSelect={handleSelect}
    />
  );
};
ImportProjectCommand.key = "project-import";
ImportProjectCommand.commandName = "Import a project";
ImportProjectCommand.sortOrder = -1;
ImportProjectCommand.useVisible = useCreateVisible;

const ExportProjectCommand: Palette.Command = (listProps) => {
  const handleExport = Project.useExport();
  const handleSelect = useCallback(() => handleExport(null), [handleExport]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Export current project"
      icon={<PProject.ExportIcon />}
      onSelect={handleSelect}
    />
  );
};
ExportProjectCommand.key = "project-export";
ExportProjectCommand.commandName = "Export current project";
ExportProjectCommand.sortOrder = -1;
ExportProjectCommand.useVisible = useViewVisible;

export const COMMANDS = [CreateCommand, ImportProjectCommand, ExportProjectCommand];
