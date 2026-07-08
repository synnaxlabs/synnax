// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { DisconnectedError, project, type Synnax } from "@synnaxlabs/client";
import { Access, Mosaic, type Pluto, type Status } from "@synnaxlabs/pluto";
import { deep, uuid } from "@synnaxlabs/x";

import { LAYOUT_FILE_NAME } from "@/feature/project/export";
import { type Import } from "@/platform/import";
import { type Layout } from "@/platform/layout";
import { Runtime } from "@/platform/runtime";
import { Session } from "@/session";

// Rewrites every reference to an imported component's original key with the key of the
// resource actually created for it. Without it the original-key tabs resolve to nothing
// and the ingesters' new-key tabs pile up as duplicates.
const remapLayoutKeys = (
  slice: Session.Layout.SliceState,
  remap: Map<string, string>,
): Session.Layout.SliceState => {
  if (remap.size === 0) return slice;
  const next = deep.copy(slice);
  next.layouts = Object.fromEntries(
    Object.entries(next.layouts).map(([key, layout]) => {
      const newKey = remap.get(key) ?? key;
      return [newKey, { ...layout, key: newKey }];
    }),
  );
  Object.values(next.mosaics).forEach((mosaic) => {
    Mosaic.forEachNode(mosaic.root, (node) => {
      node.tabs?.forEach((tab) => {
        const newKey = remap.get(tab.tabKey);
        if (newKey != null) tab.tabKey = newKey;
      });
      if (node.selected != null)
        node.selected = remap.get(node.selected) ?? node.selected;
    });
    if (mosaic.activeTab != null)
      mosaic.activeTab = remap.get(mosaic.activeTab) ?? mosaic.activeTab;
    if (mosaic.focused != null)
      mosaic.focused = remap.get(mosaic.focused) ?? mosaic.focused;
  });
  return next;
};

export const ingest: Import.DirectoryIngester = async (
  name,
  files,
  { client, fileIngesters, placeLayout, store, fluxStore },
) => {
  if (!Access.updateGranted({ id: project.TYPE_ONTOLOGY_ID, store: fluxStore, client }))
    throw new Error("You do not have permission to import projects");
  if (client == null) throw new DisconnectedError();
  const layoutData = files.find((file) => file.name === LAYOUT_FILE_NAME);
  if (layoutData == null) throw new Error(`${LAYOUT_FILE_NAME} not found`);
  const layout = Session.Layout.migrateLayout(layoutData.data);
  const projectKey = uuid.create();
  const proj: project.Project = { key: projectKey, name, layout };
  // Create the project first so imported components can be parented to it; its layout
  // is rewritten and installed below once their real keys are known.
  await client.projects.create(proj);

  const remap = new Map<string, string>();
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
    const id = await ingest(data, {
      layout: childLayout,
      placeLayout,
      store: fluxStore,
      client,
      projectKey,
    });
    if (id != null && id.key !== key) remap.set(key, id.key);
  }

  const remappedLayout = remapLayoutKeys(layout, remap);
  store.dispatch(Session.Project.select(proj.key));
  store.dispatch(Session.Layout.setProject({ slice: remappedLayout }));
  if (remap.size > 0) await client.projects.setLayout(projectKey, remappedLayout);
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
  let name: string | undefined = "project";
  handleError(async () => {
    const directory = await Runtime.pickDirectory({ title: "Import a Project" });
    if (directory == null) return;
    name = directory.name;
    const fileData = await Promise.all(
      directory.files.map(async (file): Promise<Import.File> => ({
        name: file.name,
        data: JSON.parse(await file.read()),
      })),
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
