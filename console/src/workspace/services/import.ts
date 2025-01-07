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

import { INGESTORS } from "@/ingestors";
import { Layout } from "@/layout";
import { Workspace } from "@/workspace";

interface IngestContext {
  addStatus: Status.AddStatusFn;
  client: Synnax | null;
  placeLayout: Layout.Placer;
  store: Store;
}

export const ingest = async (
  path: string,
  { addStatus, client, placeLayout, store }: IngestContext,
) => {
  try {
    const files = await readDir(path);
    const layoutInfo = files.find((file) => file.name === Workspace.LAYOUT_FILE_NAME);
    const layoutInfoPath = await join(path, Workspace.LAYOUT_FILE_NAME);
    if (layoutInfo == null) throw new Error(`${layoutInfoPath} does not exist`);
    const layoutData = await readTextFile(layoutInfoPath);
    const layout = Layout.anySliceStateZ.parse(JSON.parse(layoutData));
    await Promise.allSettled(
      Object.entries(layout.layouts).map(async ([key, layout]) => {
        const ingest = INGESTORS[layout.type];
        if (ingest == null) return;
        const data = await readTextFile(await join(path, `${key}.json`));
        placeLayout(ingest({ data, name: layout.name, store, key, layout }));
      }),
    );
    const wsKey = uuid();
    const wsName = path.split(sep()).at(-1);
    if (wsName == null) throw new Error("Workspace name not found");
    const ws: workspace.Workspace = { key: wsKey, name: wsName, layout };
    store.dispatch(Workspace.add(ws));
    store.dispatch(Layout.setWorkspace({ slice: layout, keepNav: false }));
    await client?.workspaces.create(ws);
  } catch (e) {
    if (!(e instanceof Error)) throw e;
    addStatus({
      message: "Failed to import workspace",
      description: e.message,
      variant: "error",
    });
  }
};

export const import_ = async ({
  addStatus,
  client,
  placeLayout,
  store,
}: IngestContext) => {
  const path = await open({
    title: "Import a Workspace",
    multiple: false,
    directory: true,
  });
  if (path == null) return;
  await ingest(path, { addStatus, client, placeLayout, store });
};
