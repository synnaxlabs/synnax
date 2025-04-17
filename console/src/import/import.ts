// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { type Synnax } from "@synnaxlabs/client";
import { Status, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { isTauri } from "@tauri-apps/api/core";
import { sep } from "@tauri-apps/api/path";
import { open } from "@tauri-apps/plugin-dialog";
import { readTextFile } from "@tauri-apps/plugin-fs";
import { useStore } from "react-redux";

import { NULL_CLIENT_ERROR } from "@/errors";
import { type FileIngestor } from "@/import/ingestor";
import { trimFileName } from "@/import/trimFileName";
import { Layout } from "@/layout";
import { type RootState } from "@/store";
import { Workspace } from "@/workspace";

export interface ImportArgs {
  handleError: Status.ErrorHandler;
  client: Synnax | null;
  placeLayout: Layout.Placer;
  store: Store;
  workspaceKey?: string;
}

export interface Importer {
  (args: ImportArgs): Promise<void>;
}

export interface ImporterCreator {
  (ingest: FileIngestor, type?: string): Importer;
}


async function setActiveWorkspace(store: Store, workspaceKey: string | undefined, client: Synnax | null) {
  const storeState = store.getState();
  const activeWorkspaceKey = Workspace.selectActiveKey(storeState);
  if (workspaceKey != null && activeWorkspaceKey !== workspaceKey) {
    let ws = Workspace.select(storeState, workspaceKey);
    if (ws == null) {
      if (client == null) throw NULL_CLIENT_ERROR;
      ws = await client.workspaces.retrieve(workspaceKey);
    }
    store.dispatch(Workspace.add(ws));
    store.dispatch(
      Layout.setWorkspace({ slice: ws.layout as Layout.SliceState, keepNav: false })
    );
  }
}


const FILTERS = [{ name: "JSON", extensions: ["json"] }];

const createImporterTauri: ImporterCreator =
  (ingest, type = "visualization") =>
    async ({ store, client, placeLayout, handleError, workspaceKey }) => {
      const paths = await open({
        title: `Import ${type}`,
        filters: FILTERS,
        multiple: true,
        directory: false,
      });
      if (paths == null) return;
      await setActiveWorkspace(store, workspaceKey, client);
      await Promise.allSettled(
        paths.map(async (path) => {
          try {
            const data = await readTextFile(path);
            let name = path.split(sep()).pop();
            if (name == null) throw new Error(`Cannot read file located at ${path}`);
            name = trimFileName(name);
            ingest(data, { layout: { name }, placeLayout, store });
          } catch (e) {
            handleError(e, `Failed to import ${type} at ${path}`);
          }
        }),
      );
    };

const createImporterWeb: ImporterCreator =
  (ingest, type = "visualization") =>
    async ({ store, client, placeLayout, handleError, workspaceKey }) => {
      // Select files for import by creating + clicking a hidden input[type="file"]
      function selectMultipleFiles(): Promise<FileList | null> {
        return new Promise((resolve) => {
          const input = document.createElement('input');
          input.type = 'file';
          input.multiple = true;
          input.accept = FILTERS.flatMap(f => f.extensions).join(',');
          input.style.display = 'none';
          document.body.appendChild(input);
          input.addEventListener('change', () => {
            resolve(input.files);
            document.body.removeChild(input);
          });
          input.click();
        });
      }

      const paths = await selectMultipleFiles();
      if (paths == null) return;

      await setActiveWorkspace(store, workspaceKey, client);

      await Promise.allSettled(
        Array.from(paths).map(async (file) => {
          try {
            const data = await file.text();
            let name = file.name;
            if (name == null) throw new Error(`Cannot read file located at ${file.name}`);
            name = trimFileName(name);
            ingest(data, { layout: { name }, placeLayout, store });
          } catch (e) {
            handleError(e, `Failed to import ${type} at ${file.name}`)
          }
        }),
      );
    };

export const createImporter = (ingest: FileIngestor, type?: string) => {
  if (isTauri()) return createImporterTauri(ingest, type);
  return createImporterWeb(ingest, type);
};

export const use = (
  import_: Importer,
  workspaceKey?: string,
): (() => Promise<void>) => {
  const placeLayout = Layout.usePlacer();
  const store = useStore<RootState>();
  const client = PSynnax.use();
  const handleError = Status.useErrorHandler();
  return () => import_({ store, placeLayout, client, handleError, workspaceKey });
};
