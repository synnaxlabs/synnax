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
  type ontology,
  project,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { ingestBatch } from "@/platform/import/ingestBatch";
import { Panel } from "@/platform/panel";
import { Runtime } from "@/platform/runtime";
import { Session } from "@/session";

export interface FileIngesterContext {
  client: Client | null;
  projectKey: project.Key;
  /**
   * The name of the file the data was read from, extension included. The Core names
   * the resource after the file when the file's contents carry no name.
   */
  fileName: string;
}

/**
 * Creates the resource the data describes and returns its ID. Opening a tab for it
 * belongs to the caller, which decides where it lands.
 */
export interface FileIngester {
  (
    data: unknown,
    ctx: FileIngesterContext,
  ): void | ontology.ID | Promise<void | ontology.ID>;
}

interface BundleIngesterContext {
  client: Client | null;
  store: Store;
}

/**
 * Imports a zipped bundle read from a picked or dropped source. name is the source
 * archive or folder's name, the Core's fallback for naming the imported resource.
 */
export interface BundleIngester {
  (
    name: string,
    bundle: Uint8Array<ArrayBuffer>,
    ctx: BundleIngesterContext,
  ): Promise<void>;
}

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
      extension: "json",
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
      // ingestBatch names failures after the item, so the path doubles as the name.
      items: files.map((file) => ({ name: file.path, readBytes: file.readBytes })),
      ingest: async (file) =>
        await ingestServer(
          JSON.parse(new TextDecoder().decode(await file.readBytes())),
          { client, projectKey: activeProjectKeyAfter, fileName: file.name },
        ),
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
