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

import { type Import } from "@/import";
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
  const layout = Layout.migrateSlice(Layout.anySliceStateZ.parse(layoutData.data));
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

  for (const [key, childLayout] of Object.entries(layout.layouts)) {
    const ingest = fileIngesters[childLayout.type];
    if (ingest == null) continue;
    const data = files.find(
      (file) =>
        file.name === `${childLayout.name}.json` ||
        file.name === `${key}.json` ||
        (typeof file.data === "object" &&
          file.data != null &&
          (("key" in file.data && file.data.key === key) ||
            ("name" in file.data && file.data.name === childLayout.name))),
    )?.data;
    if (data == null) throw new Error(`Data for ${key} not found`);
    await ingest(data, {
      layout: childLayout,
      placeLayout,
      store: fluxStore,
      client,
      workspaceKey: wsKey,
    });
  }
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
    const directory = await Runtime.pickDirectory({ title: "Import a Workspace" });
    if (directory == null) return;
    name = directory.name;
    const fileData = await Promise.all(
      directory.files.map(
        async (file): Promise<Import.File> => ({
          name: file.name,
          data: JSON.parse(await file.read()),
        }),
      ),
    );
    await ingest(name, fileData, {
      client,
      fileIngesters,
      placeLayout,
      store,
      fluxStore,
    });
  }, `Failed to import ${name}`);
};
