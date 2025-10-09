// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { type Synnax, type workspace } from "@synnaxlabs/client";
import { type Status } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { join, sep } from "@tauri-apps/api/path";
import { open } from "@tauri-apps/plugin-dialog";
import { readDir, readTextFile } from "@tauri-apps/plugin-fs";

import { type Import } from "@/import";
import { Layout } from "@/layout";
import { Runtime } from "@/runtime";
import { Workspace } from "@/workspace";

export const ingest: Import.DirectoryIngestor = async (
  name,
  files,
  { client, fileIngestors, placeLayout, store },
) => {
  const layoutData = files.find((file) => file.name === Workspace.LAYOUT_FILE_NAME);
  if (layoutData == null) throw new Error(`${Workspace.LAYOUT_FILE_NAME} not found`);
  const layout = Layout.migrateSlice(JSON.parse(layoutData.data));
  const wsKey = uuid.create();
  const wsName = name;
  const ws: workspace.Workspace = { key: wsKey, name: wsName, layout };
  const createdWs = await client?.workspaces.create(ws);
  store.dispatch(Workspace.add(createdWs ?? ws));
  store.dispatch(
    Layout.setWorkspace({
      slice: (createdWs?.layout as Layout.SliceState) ?? layout,
      keepNav: false,
    }),
  );
  Object.entries(layout.layouts).forEach(([key, layout]) => {
    const ingest = fileIngestors[layout.type];
    if (ingest == null) return;
    const data = files.find((file) => file.name === `${key}.json`)?.data;
    if (data == null) throw new Error(`Data for ${key} not found`);
    ingest(data, { layout, placeLayout, store });
  });
};

export interface IngestContext {
  handleError: Status.ErrorHandler;
  client: Synnax | null;
  fileIngestors: Import.FileIngestors;
  placeLayout: Layout.Placer;
  store: Store;
}

export const import_ = ({
  handleError,
  client,
  fileIngestors,
  placeLayout,
  store,
}: IngestContext) => {
  let name: string | undefined = "workspace";
  handleError(async () => {
    if (Runtime.ENGINE !== "tauri")
      throw new Error(
        "Cannot import items from a dialog when running Synnax in the browser.",
      );
    const path = await open({
      title: "Import a Workspace",
      multiple: false,
      directory: true,
    });
    if (path == null) return;
    name = path.split(sep()).at(-1);
    if (name == null) throw new Error("Cannot read workspace");
    const files = await readDir(path);
    const fileData = await Promise.all(
      files.map(async (file) => ({
        name: file.name,
        data: await readTextFile(await join(path, file.name)),
      })),
    );
    await ingest(name, fileData, { client, fileIngestors, placeLayout, store });
  }, `Failed to import ${name}`);
};
