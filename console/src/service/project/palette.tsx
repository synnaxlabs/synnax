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

import { Palette } from "@/component/palette";
import { Project } from "@/component/project";
import { useFileIngesters } from "@/service/import/FileIngestersProvider";
import { import_ } from "@/service/project/import";
import { useExport } from "@/service/project/export";

const useCreateVisible = () => Access.useCreateGranted(project.TYPE_ONTOLOGY_ID);
const useViewVisible = () => Access.useRetrieveGranted(project.TYPE_ONTOLOGY_ID);

const CreateCommand = Palette.createCommand({
  key: "project_create",
  name: "Create a project",
  icon: <PProject.CreateIcon />,
  useOnSelect: Project.useCreateModal,
  useVisible: useCreateVisible,
});

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

const useExportCurrentProject = (): (() => void) => {
  const handleExport = useExport();
  return useCallback(() => handleExport(null), [handleExport]);
};

const ExportProjectCommand = Palette.createCommand({
  key: "project_export",
  name: "Export current project",
  icon: <PProject.ExportIcon />,
  useOnSelect: useExportCurrentProject,
  useVisible: useViewVisible,
  sortOrder: -1,
});

export const COMMANDS = [CreateCommand, ImportProjectCommand, ExportProjectCommand];
