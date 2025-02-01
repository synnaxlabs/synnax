// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { type Synnax, type workspace } from "@synnaxlabs/client";
import { type Status } from "@synnaxlabs/pluto";
import { join, sep } from "@tauri-apps/api/path";
import { open } from "@tauri-apps/plugin-dialog";
import { readDir, readTextFile } from "@tauri-apps/plugin-fs";
import { v4 as uuid } from "uuid";

import { type Import } from "@/import";
import { Layout } from "@/layout";
import { Workspace } from "@/workspace";

export const ingest: Import.DirectoryIngestor = async (
  name,
  files,
  { client, ingestors, placeLayout, store },
) => {
  const layoutData = files.find((file) => file.name === Workspace.LAYOUT_FILE_NAME);
  if (layoutData == null) throw new Error(`${Workspace.LAYOUT_FILE_NAME} not found`);
  const layout = Layout.anySliceStateZ.parse(JSON.parse(layoutData.data));
  Object.entries(layout.layouts).forEach(([key, layout]) => {
    const ingest = ingestors[layout.type];
    if (ingest == null) return;
    const data = files.find((file) => file.name === `${key}.json`)?.data;
    if (data == null) throw new Error(`Data for ${key} not found`);
    ingest(data, { layout, placeLayout, store });
  });
  const wsKey = uuid();
  const wsName = name;
  const ws: workspace.Workspace = { key: wsKey, name: wsName, layout };
  store.dispatch(Workspace.add(ws));
  store.dispatch(Layout.setWorkspace({ slice: layout, keepNav: false }));
  await client?.workspaces.create(ws);
};

export interface IngestContext {
  handleException: Status.HandleExcFn;
  client: Synnax | null;
  ingestors: Record<string, Import.FileIngestor>;
  placeLayout: Layout.Placer;
  store: Store;
}

export const import_ = async ({
  handleException,
  client,
  ingestors,
  placeLayout,
  store,
}: IngestContext) => {
  const path = await open({
    title: "Import a Workspace",
    multiple: false,
    directory: true,
  });
  if (path == null) return;
  try {
    const name = path.split(sep()).at(-1);
    if (name == null) throw new Error("Cannot read workspace");
    const files = await readDir(path);
    const fileData = await Promise.all(
      files.map(async (file) => ({
        name: file.name,
        data: await readTextFile(await join(path, file.name)),
      })),
    );
    await ingest(name, fileData, { client, ingestors, placeLayout, store });
  } catch (e) {
    handleException(e, "Failed to import workspace");
  }
};
