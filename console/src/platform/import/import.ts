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
import { Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { type FileIngester } from "@/platform/import/ingester";
import { Panel } from "@/platform/panel";
import { Runtime } from "@/platform/runtime";
import { Session } from "@/session";

/**
 * Imports data by streaming its bytes to the Core, which owns envelope decoding, type
 * resolution for typeless legacy Console states, legacy-version migration, file-name
 * naming, and project parenting. Opens the created resource as a tab.
 * @throws {DisconnectedError} if no cluster is connected.
 */
export const ingestServer: FileIngester = async (
  data,
  { openTab, client, projectKey, fileName },
) => {
  if (client == null) throw new DisconnectedError();
  const id = await client.imex.import(JSON.stringify(data), {
    encoding: "JSON",
    fileName,
    parent: project.ontologyID(projectKey),
  });
  openTab({ variant: "resource", resource: id });
  return id;
};

const FILTERS = [{ name: "JSON", extensions: ["json"] }];

interface ImportComponentParams {
  handleError: Status.ErrorHandler;
  client: Client | null;
  openTab: Panel.OpenTab;
  store: Store;
  projectKey?: string;
}

const importComponent = ({
  store,
  client,
  openTab,
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
    files.forEach((file) =>
      handleError(async () => {
        const data = await file.read();
        await ingestServer(JSON.parse(data), {
          openTab,
          client,
          projectKey: activeProjectKeyAfter,
          fileName: file.name,
        });
      }, `Failed to import ${file.name}`),
    );
  });
};

export const useImport = (): ((projectKey?: string) => void) => {
  const openTab = Panel.useOpenTab();
  const store = Session.useStore();
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  return useCallback(
    (projectKey?: string) =>
      importComponent({ store, openTab, client, handleError, projectKey }),
    [store, openTab, client, handleError],
  );
};
