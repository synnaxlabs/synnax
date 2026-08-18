// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, imex, project } from "@synnaxlabs/client";
import { Access, Project as PProject, Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Command } from "@/platform/command";
import { Export } from "@/platform/export";
import { Import } from "@/platform/import";
import { Project } from "@/platform/project";
import { Session } from "@/session";

const useCreateVisible = () => Access.useCreateGranted(project.TYPE_ONTOLOGY_ID);

const CreateCommand = Command.create({
  key: "project_create",
  name: "Create project",
  icon: <PProject.CreateIcon />,
  useOnSelect: Project.useCreateModal,
  useVisible: useCreateVisible,
});

/** A modal that imports a project from a .zip export or its extracted folder. */
const useImportModal = Import.createModal({
  header: "Project.Import",
  resourceName: "project",
  useOnImport: () => {
    const client = Synnax.use();
    const store = Session.useStore();
    return useCallback(
      async (bundle: Uint8Array<ArrayBuffer>, fileName: string) => {
        if (client == null) throw new DisconnectedError();
        const imported = await client.projects.import(bundle, { fileName });
        store.dispatch(Session.Project.select(imported.key));
        return imported.name;
      },
      [client, store],
    );
  },
});

const ImportProjectCommand = Command.create({
  key: "project_import",
  name: "Import project",
  icon: <PProject.ImportIcon />,
  useOnSelect: useImportModal,
  useVisible: useCreateVisible,
});

const ExportProjectCommand = Command.create({
  key: "project_export",
  name: "Export current project",
  icon: <PProject.ExportIcon />,
  useOnSelect: () => {
    const handleExport = Export.use();
    const getSelected = Session.Project.useGetSelected();
    const client = Synnax.use();
    const handleError = Status.useErrorHandler();
    return useCallback(() => {
      let name = "project";
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
        const p = await client.projects.retrieve(getSelected());
        name = p.name;
        handleExport({
          stream: (client) => client.projects.export(p.key, imex.JSON_OPTIONS),
          name,
          extension: "zip",
        });
      }, `Failed to export ${name}`);
    }, [handleError, client, getSelected, handleExport]);
  },
  useVisible: () => Access.useRetrieveGranted(project.TYPE_ONTOLOGY_ID),
});

export const COMMANDS = [CreateCommand, ImportProjectCommand, ExportProjectCommand];
