// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { project } from "@synnaxlabs/client";
import {
  Access,
  Flux,
  type Pluto,
  Project as PProject,
  Status,
  Synnax,
} from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { useExport } from "@/feature/project/export";
import { import_ } from "@/feature/project/import";
import { Command } from "@/platform/command";
import { Import } from "@/platform/import";
import { Layout } from "@/platform/layout";
import { Project } from "@/platform/project";
import { Session } from "@/session";

const useCreateVisible = () => Access.useCreateGranted(project.TYPE_ONTOLOGY_ID);
const useViewVisible = () => Access.useRetrieveGranted(project.TYPE_ONTOLOGY_ID);

const CreateCommand = Command.create({
  key: "project_create",
  name: "Create a project",
  icon: <PProject.CreateIcon />,
  useOnSelect: Project.useCreateModal,
  useVisible: useCreateVisible,
});

const useImportProject = () => {
  const placeLayout = Layout.usePlacer();
  const handleError = Status.useErrorHandler();
  const store = Session.useStore();
  const client = Synnax.use();
  const fluxStore = Flux.useStore<Pluto.FluxStore>();
  const fileIngesters = Import.useFileIngesters();
  return useCallback(
    () =>
      import_({ placeLayout, handleError, store, client, fluxStore, fileIngesters }),
    [placeLayout, handleError, store, client, fluxStore, fileIngesters],
  );
};

const ImportProjectCommand = Command.create({
  key: "project_import",
  name: "Import a project",
  icon: <PProject.ImportIcon />,
  useOnSelect: useImportProject,
  useVisible: useCreateVisible,
  sortOrder: -1,
});

const useExportCurrentProject = (): (() => void) => {
  const handleExport = useExport();
  return useCallback(() => handleExport(null), [handleExport]);
};

const ExportProjectCommand = Command.create({
  key: "project_export",
  name: "Export current project",
  icon: <PProject.ExportIcon />,
  useOnSelect: useExportCurrentProject,
  useVisible: useViewVisible,
  sortOrder: -1,
});

export const COMMANDS = [CreateCommand, ImportProjectCommand, ExportProjectCommand];
