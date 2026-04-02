// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { type Synnax, workspace } from "@synnaxlabs/client";
import { Access, type Pluto, type Status } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { join, sep } from "@tauri-apps/api/path";
import { open } from "@tauri-apps/plugin-dialog";
import { readDir, readTextFile } from "@tauri-apps/plugin-fs";

import { Import } from "@/import";
import { Layout } from "@/layout";
import { Runtime } from "@/runtime";
import { Workspace } from "@/workspace";

export const ingest: Import.DirectoryIngester = async (
  name,
  files,
  { client, fileIngesters, placeLayout, store, fluxStore },
) => {
  if (
    !Access.updateGranted({ id: workspace.TYPE_ONTOLOGY_ID, store: fluxStore, client })
  )
    throw new Error("You do not have permission to import workspaces");
  const layoutData = files.find((file) => file.name === Workspace.LAYOUT_FILE_NAME);
  if (layoutData == null) throw new Error(`${Workspace.LAYOUT_FILE_NAME} not found`);
  const parsed = Layout.migrateSlice(Layout.anySliceStateZ.parse(layoutData.data));
  const { slice: layout, oldKeyForNew } = Layout.remapKeys(parsed);
  const wsKey = uuid.create();
  const wsName = name;
  const ws: workspace.Workspace = { key: wsKey, name: wsName, layout };
  const createdWs = await client?.workspaces.create(ws);
  store.dispatch(Workspace.setActive(createdWs ?? ws));
  store.dispatch(
    Layout.setWorkspace({
      slice: (createdWs?.layout as Layout.SliceState) ?? layout,
      keepNav: false,
    }),
  );

  Object.entries(layout.layouts).forEach(([newKey, layoutEntry]) => {
    const ingest = fileIngesters[layoutEntry.type];
    if (ingest == null) return;
    // File lookup uses the original exported key/name since the files on
    // disk still contain the pre-remap keys.
    const oldKey = oldKeyForNew.get(newKey) ?? newKey;
    const data = files.find(
      (file) =>
        file.name === `${layoutEntry.name}.json` ||
        file.name === `${oldKey}.json` ||
        (typeof file.data === "object" &&
          file.data != null &&
          (("key" in file.data && file.data.key === oldKey) ||
            ("name" in file.data && file.data.name === layoutEntry.name))),
    )?.data;
    if (data == null) throw new Error(`Data for ${newKey} not found`);
    ingest(data, {
      layout: layoutEntry,
      placeLayout,
      store: fluxStore,
      client,
    });
  });
};

export interface IngestContext {
  handleError: Status.ErrorHandler;
  client: Synnax | null;
  fileIngesters: Import.FileIngesters;
  placeLayout: Layout.Placer;
  store: Store;
  fluxStore: Pluto.FluxStore;
}

export const import_ = ({
  handleError,
  client,
  fileIngesters,
  placeLayout,
  store,
  fluxStore,
}: IngestContext) => {
  let name: string | undefined = "workspace";
  handleError(async () => {
    let fileData: Import.File[];

    if (Runtime.ENGINE === "tauri") {
      const path = await open({
        title: "Import a Workspace",
        multiple: false,
        directory: true,
      });
      if (path == null) return;
      name = path.split(sep()).at(-1);
      if (name == null) throw new Error("Cannot read workspace");
      const files = await readDir(path);
      fileData = await Promise.all(
        files.map(
          async (file): Promise<Import.File> => ({
            name: file.name,
            data: JSON.parse(await readTextFile(await join(path, file.name))),
          }),
        ),
      );
    } else {
      const files = await Import.pickDirectoryFromBrowser();
      if (files == null || files.length === 0) return;
      name = files[0].webkitRelativePath.split("/")[0];
      if (!name) throw new Error("Cannot read workspace");
      fileData = await Promise.all(
        files.map(async (file): Promise<Import.File> => {
          const text = await file.text();
          return { name: file.name, data: JSON.parse(text) };
        }),
      );
    }

    await ingest(name, fileData, {
      client,
      fileIngesters,
      placeLayout,
      store,
      fluxStore,
    });
  }, `Failed to import ${name}`);
};
