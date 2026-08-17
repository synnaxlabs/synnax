// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { DisconnectedError, project, type Synnax as Client } from "@synnaxlabs/client";
import { Access, Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { ingestBatch } from "@/platform/import/ingestBatch";
import { type FileIngester } from "@/platform/import/ingester";
import { Panel } from "@/platform/panel";
import { Runtime } from "@/platform/runtime";
import { Session } from "@/session";

/**
 * Imports data by streaming its bytes to the Core, which owns envelope decoding, type
 * resolution for typeless legacy Console states, legacy-version migration, file-name
 * naming, and project parenting.
 * @throws {DisconnectedError} if no cluster is connected.
 */
export const ingestServer: FileIngester = async (
  data,
  { client, projectKey, fileName },
) => {
  if (client == null) throw new DisconnectedError();
  return await client.imex.import(JSON.stringify(data), {
    encoding: "JSON",
    fileName,
    parent: project.ontologyID(projectKey),
  });
};

const FILTERS = [{ name: "JSON", extensions: ["json"] }];

interface ImportComponentParams {
  handleError: Status.ErrorHandler;
  client: Client | null;
  openTabs: Panel.OpenTabs;
  store: Store;
  projectKey?: string;
}

const importComponent = ({
  store,
  client,
  openTabs,
  handleError,
  projectKey,
}: ImportComponentParams): void => {
  handleError(async () => {
    const files = await Runtime.pickFiles({
      title: "Import",
      filters: FILTERS,
      multiple: true,
    });
    if (files == null) return;
    const storeState = store.getState();
    const activeProjectKey = Session.Project.selectSelected(storeState);
    if (projectKey != null && activeProjectKey !== projectKey) {
      if (client == null) throw new DisconnectedError();
      const proj = await client.projects.retrieve(projectKey);
      store.dispatch(Session.Project.select(proj.key));
    }
    const activeProjectKeyAfter = Session.Project.selectSelected(store.getState());
    await ingestBatch({
      items: files,
      ingest: async (file) =>
        await ingestServer(JSON.parse(await file.read()), {
          client,
          projectKey: activeProjectKeyAfter,
          fileName: file.name,
        }),
      handleError,
      openTabs,
    });
  });
};

export const useImport = (): ((projectKey?: string) => void) => {
  const openTabs = Panel.useOpenTabs();
  const store = Session.useStore();
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  return useCallback(
    (projectKey?: string) =>
      importComponent({ store, openTabs, client, handleError, projectKey }),
    [store, openTabs, client, handleError],
  );
};

/**
 * Reports whether the subject may import into the selected project. The Core enforces
 * create on the imported resource's own type as well, which no caller can know before
 * the file is read, so a permitted import can still be refused on that second check.
 * With no project selected there is no parent to read the grant on, and the import
 * itself has nowhere to land, so the answer is deferred to the Core.
 */
export const useCanImport = (): boolean => {
  const selected = Session.Project.useSelectOptionalSelected();
  const granted = Access.useUpdateGranted(
    selected == null ? project.TYPE_ONTOLOGY_ID : project.ontologyID(selected),
  );
  return selected == null || granted;
};
