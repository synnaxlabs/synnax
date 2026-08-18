// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import {
  DisconnectedError,
  imex,
  project,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
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
 * @throws {DisconnectedError} if no Core is connected.
 */
export const ingestServer: FileIngester = async (
  data,
  { client, projectKey, fileName },
) => {
  if (client == null) throw new DisconnectedError();
  return await client.imex.import(JSON.stringify(data), {
    ...imex.JSON_OPTIONS,
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
