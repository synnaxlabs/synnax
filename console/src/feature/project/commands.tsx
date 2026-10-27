// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, imex, project } from "@synnaxlabs/client";
import { type UploadBody } from "@synnaxlabs/freighter";
import { Status } from "@synnaxlabs/lyra/status";
import { Access, Project as PProject, Synnax } from "@synnaxlabs/pluto";

import { useOpenSample } from "@/feature/project/useOpenSample";
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

const useOnImport = () => {
  const client = Synnax.use();
  const store = Session.useStore();
  return async (bundle: UploadBody, fileName: string) => {
    if (client == null) throw new DisconnectedError();
    const imported = await client.projects.import(bundle, { fileName });
    store.dispatch(Session.Project.select(imported.key));
    return imported.name;
  };
};

/** A modal that imports a project from a .zip export or its extracted folder. */
const useImportModal = Import.createModal({
  header: "Project.Import",
  resourceName: "project",
  useOnImport,
});

const ImportProjectCommand = Command.create({
  key: "project_import",
  name: "Import project",
  icon: <PProject.ImportIcon />,
  useOnSelect: useImportModal,
  useVisible: useCreateVisible,
});

const useExportSelected = () => {
  const handleExport = Export.use();
  const getSelected = Session.Project.useGetSelected();
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  return () => {
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      const p = await client.projects.retrieve(getSelected());
      handleExport({
        stream: (client) => client.projects.export(p.key, imex.JSON_OPTIONS),
        name: p.name,
        extension: "zip",
      });
    }, "Failed to export project");
  };
};

const ExportProjectCommand = Command.create({
  key: "project_export",
  name: "Export current project",
  icon: <PProject.ExportIcon />,
  useOnSelect: useExportSelected,
  useVisible: () => Access.useRetrieveGranted(project.TYPE_ONTOLOGY_ID),
});

const useOpenSampleVisible = (): boolean => {
  const granted = useCreateVisible();
  return Session.Runtime.CORE_EMBEDDED && granted;
};

const OpenSampleCommand = Command.create({
  key: "project_open_sample",
  name: "Open sample project",
  icon: <PProject.CreateIcon />,
  useOnSelect: useOpenSample,
  useVisible: useOpenSampleVisible,
});

export const COMMANDS = [
  CreateCommand,
  ImportProjectCommand,
  ExportProjectCommand,
  OpenSampleCommand,
];
