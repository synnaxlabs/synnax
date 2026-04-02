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

import { useFileIngesters } from "@/import/FileIngestersProvider";
import { ingestComponent } from "@/import/ingestComponent";
import { type FileIngesters } from "@/import/ingester";
import { trimFileName } from "@/import/trimFileName";
import { Layout } from "@/layout";
import { Runtime } from "@/runtime";
import { type RootState } from "@/store";
import { Workspace } from "@/workspace";

interface PickFromBrowserOpts {
  accept?: string;
  multiple?: boolean;
  directory?: boolean;
}

const pickFromBrowser = (opts: PickFromBrowserOpts): Promise<File[] | null> =>
  new Promise((resolve) => {
    const input = document.createElement("input");
    input.type = "file";
    if (opts.accept != null) input.accept = opts.accept;
    if (opts.multiple === true) input.multiple = true;
    if (opts.directory === true) input.webkitdirectory = true;
    input.addEventListener("cancel", () => resolve(null));
    input.onchange = () =>
      resolve(input.files != null ? Array.from(input.files) : null);
    input.click();
  });

export const pickFilesFromBrowser = (
  accept: string,
  multiple: boolean,
): Promise<File[] | null> => pickFromBrowser({ accept, multiple });

export const pickDirectoryFromBrowser = (): Promise<File[] | null> =>
  pickFromBrowser({ directory: true });

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

    if (Runtime.ENGINE === "tauri") {
      const paths = await open({
        title: "Import",
        filters: FILTERS,
        multiple: true,
        directory: false,
      });
      if (paths == null) return;
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
    } else {
      const files = await pickFilesFromBrowser(".json", true);
      if (files == null || files.length === 0) return;
      for (const file of files)
        handleError(async () => {
          const text = await file.text();
          const name = trimFileName(file.name);
          ingestComponent(JSON.parse(text), name, fileIngesters, {
            layout: { name },
            placeLayout,
            store: fluxStore,
            client,
          });
        }, `Failed to import ${file.name}`);
    }
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
