// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { project } from "@synnaxlabs/client";
import { Access, Project as PProject, Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { import_ } from "@/feature/project/import";
import { Command } from "@/platform/command";
import { Export } from "@/platform/export";
import { Project } from "@/platform/project";
import { Session } from "@/session";

const useCreateVisible = () => Access.useCreateGranted(project.TYPE_ONTOLOGY_ID);

const CreateCommand = Command.create({
  key: "project_create",
  name: "Create a project",
  icon: <PProject.CreateIcon />,
  useOnSelect: Project.useCreateModal,
  useVisible: useCreateVisible,
});

const useImportProject = () => {
  const handleError = Status.useErrorHandler();
  const store = Session.useStore();
  const client = Synnax.use();
  return useCallback(
    () => import_({ handleError, store, client }),
    [handleError, store, client],
  );
};

const ImportProjectCommand = Command.create({
  key: "project_import",
  name: "Import a project",
  icon: <PProject.ImportIcon />,
  useOnSelect: useImportProject,
  useVisible: useCreateVisible,
});

const ExportProjectCommand = Command.create({
  key: "project_export",
  name: "Export current project",
  icon: <PProject.ExportIcon />,
  useOnSelect: () => {
    const handleExport = Export.use();
    const getSelected = Session.Project.useGetSelected();
    return useCallback(() => {
      // getSelected throws when no project is selected, so it runs inside the stream
      // fetcher, within the export handler's error boundary.
      handleExport({
        stream: (client) => client.projects.export(getSelected()),
        name: "project",
        extension: "zip",
      });
    }, [handleExport, getSelected]);
  },
  useVisible: () => Access.useRetrieveGranted(project.TYPE_ONTOLOGY_ID),
});

export const COMMANDS = [CreateCommand, ImportProjectCommand, ExportProjectCommand];
