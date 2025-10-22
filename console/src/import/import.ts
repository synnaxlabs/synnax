// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { DisconnectedError, type Synnax as Client } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { sep } from "@tauri-apps/api/path";
import { open } from "@tauri-apps/plugin-dialog";
import { readTextFile } from "@tauri-apps/plugin-fs";
import { useCallback } from "react";
import { useStore } from "react-redux";

import { type FileIngestor } from "@/import/ingestor";
import { trimFileName } from "@/import/trimFileName";
import { Layout } from "@/layout";
import { Runtime } from "@/runtime";
import { type RootState } from "@/store";
import { Workspace } from "@/workspace";

export interface ImportArgs {
  handleError: Status.ErrorHandler;
  client: Client | null;
  placeLayout: Layout.Placer;
  store: Store;
  workspaceKey?: string;
}

export interface Importer {
  (args: ImportArgs): void;
}

export interface ImporterCreator {
  (ingest: FileIngestor, type: string): Importer;
}

const FILTERS = [{ name: "JSON", extensions: ["json"] }];

export const createImporter: ImporterCreator =
  (ingest, type) =>
  ({ store, client, placeLayout, handleError, workspaceKey }) => {
    handleError(async () => {
      if (Runtime.ENGINE !== "tauri")
        throw new Error(
          "Cannot import items from a dialog when running Synnax in the browser.",
        );
      const paths = await open({
        title: `Import ${type}`,
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
          ingest(JSON.parse(data), { layout: { name }, placeLayout, store });
        }, `Failed to import ${type} at ${path}`),
      );
    });
  };

export const use = (import_: Importer, workspaceKey?: string): (() => void) => {
  const placeLayout = Layout.usePlacer();
  const store = useStore<RootState>();
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  return useCallback(
    () => import_({ store, placeLayout, client, handleError, workspaceKey }),
    [import_, store, placeLayout, client, handleError, workspaceKey],
  );
};
