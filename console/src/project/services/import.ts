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
import { Access, type Pluto, type Status } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";

import { type Import } from "@/import";
import { Layout } from "@/layout";
import { Project } from "@/project";
import { Runtime } from "@/runtime";

// Swaps imported themes for the current defaults before validation. A stale theme blob
// from an older export must not fail anySliceStateZ and block the import.
const stripThemes = (data: unknown): unknown => {
  if (typeof data !== "object" || data == null) return data;
  return {
    ...data,
    themes: Layout.ZERO_SLICE_STATE.themes,
    activeTheme: Layout.ZERO_SLICE_STATE.activeTheme,
  };
};

export const ingest: Import.DirectoryIngester = async (
  name,
  files,
  { client, fileIngesters, placeLayout, store, fluxStore },
) => {
  if (!Access.updateGranted({ id: project.TYPE_ONTOLOGY_ID, store: fluxStore, client }))
    throw new Error("You do not have permission to import projects");
  if (client == null) throw new DisconnectedError();
  const layoutData = files.find((file) => file.name === Project.LAYOUT_FILE_NAME);
  if (layoutData == null) throw new Error(`${Project.LAYOUT_FILE_NAME} not found`);
  // Parse the legacy layout blob for child-resource ingest (each child layout
  // points at a viz JSON to import). The project's tiling no longer installs
  // from the blob; a panel-aware import flow lands with the export/import
  // rewrite.
  const layout = Layout.migrateSlice(
    Layout.anySliceStateZ.parse(stripThemes(layoutData.data)),
  );
  const projectKey = uuid.create();
  const proj: project.Project = { key: projectKey, name, layout: {} };
  // Create the project first so imported components can be parented to it.
  await client.projects.create(proj);
  store.dispatch(Project.setActive({ key: projectKey, name }));

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
      projectKey,
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
  let name: string | undefined = "project";
  handleError(async () => {
    const directory = await Runtime.pickDirectory({ title: "Import a Project" });
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
