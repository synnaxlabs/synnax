// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { DisconnectedError, type Synnax as Client } from "@synnaxlabs/client";
import { Flux, type Pluto, Status, Synnax } from "@synnaxlabs/pluto";
import { sep } from "@tauri-apps/api/path";
import { open } from "@tauri-apps/plugin-dialog";
import { readTextFile } from "@tauri-apps/plugin-fs";
import { useCallback } from "react";
import { useStore } from "react-redux";
import { ZodError } from "zod";

import { useFileIngesters } from "@/import/FileIngestersProvider";
import { type FileIngesterContext, type FileIngesters } from "@/import/ingester";
import { trimFileName } from "@/import/trimFileName";
import { Layout } from "@/layout";
import { Runtime } from "@/runtime";
import { type RootState } from "@/store";
import { Workspace } from "@/workspace";

export const ingestComponent = (
  data: unknown,
  fileName: string,
  fileIngesters: FileIngesters,
  ctx: FileIngesterContext,
): void => {
  for (const ingest of Object.values(fileIngesters))
    try {
      ingest(data, ctx);
      return;
    } catch (e) {
      if (e instanceof ZodError) continue;
      else throw e;
    }
  throw new Error(`${fileName} cannot be imported.`);
};

const FILTERS = [{ name: "JSON", extensions: ["json"] }];

interface ImportComponentArgs {
  handleError: Status.ErrorHandler;
  client: Client | null;
  placeLayout: Layout.Placer;
  store: Store;
  workspaceKey?: string;
  fluxStore: Pluto.FluxStore;
  fileIngesters: FileIngesters;
}

const importComponent = ({
  store,
  client,
  placeLayout,
  handleError,
  workspaceKey,
  fluxStore,
  fileIngesters,
}: ImportComponentArgs): void => {
  handleError(async () => {
    if (Runtime.ENGINE !== "tauri")
      throw new Error(
        "Cannot import components from a dialog when running Synnax in the browser.",
      );
    const paths = await open({
      title: "Import",
      filters: FILTERS,
      multiple: true,
      directory: false,
    });
    if (paths == null) return;
    const storeState = store.getState();
    const activeWorkspaceKey = Workspace.selectActiveKey(storeState);
    if (workspaceKey != null && activeWorkspaceKey !== workspaceKey) {
      if (client == null) throw new DisconnectedError();
      const ws = await client.workspaces.retrieve(workspaceKey);
      store.dispatch(Workspace.setActive(ws));
      store.dispatch(
        Layout.setWorkspace({
          slice: ws.layout as Layout.SliceState,
          keepNav: false,
        }),
      );
    }
    paths.forEach((path) =>
      handleError(async () => {
        const data = await readTextFile(path);
        const fileName = path.split(sep()).pop();
        if (fileName == null) throw new Error(`Cannot read file located at ${path}`);
        const name = trimFileName(fileName);
        ingestComponent(JSON.parse(data), name, fileIngesters, {
          layout: { name },
          placeLayout,
          store: fluxStore,
          client,
        });
      }, `Failed to import ${path}`),
    );
  });
};

export const useImport = (): ((workspaceKey?: string) => void) => {
  const placeLayout = Layout.usePlacer();
  const store = useStore<RootState>();
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const fluxStore = Flux.useStore<Pluto.FluxStore>();
  const fileIngesters = useFileIngesters();
  return useCallback(
    (workspaceKey?: string) =>
      importComponent({
        store,
        placeLayout,
        client,
        handleError,
        workspaceKey,
        fluxStore,
        fileIngesters,
      }),
    [store, placeLayout, client, handleError, fluxStore, fileIngesters],
  );
};
