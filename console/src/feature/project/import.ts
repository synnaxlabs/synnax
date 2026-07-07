// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import {
  DisconnectedError,
  type ontology,
  panel,
  project,
  type Synnax,
} from "@synnaxlabs/client";
import { Access, type Pluto, type Status } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { z } from "zod";

import { PANELS_FILE_NAME } from "@/feature/project/export";
import { Import } from "@/platform/import";
import { type Panel } from "@/platform/panel";
import { Runtime } from "@/platform/runtime";
import { Session } from "@/session";

/** The tiling file inside a legacy (layout-slice era) project export directory. */
export const LAYOUT_FILE_NAME = "LAYOUT.json";

const legacyLayoutZ = z.object({
  key: z.string(),
  type: z.string(),
  name: z.string(),
});

// Legacy console-state exports carried the full layout slice; only the layout
// records are needed to locate and type each component file.
const legacySliceZ = z.object({
  layouts: z.record(z.string(), legacyLayoutZ),
});

// Imported panels carry the project's tab placement, so component ingesters must
// not also open tabs in the current window.
const noopOpenTab: Panel.OpenTab = () => {};

// Rewrites every panel-tab reference to an imported component's original key with
// the ontology ID of the resource actually created for it. Without it the imported
// panels' tabs resolve to resources that only exist in the source cluster.
const remapNode = (node: panel.Node, remap: Map<string, ontology.ID>): panel.Node => {
  if (node.variant === "leaf")
    return {
      ...node,
      tabs: node.tabs.map((tab) => {
        if (tab.variant !== "resource") return tab;
        const next = remap.get(tab.resource.key);
        return next == null ? tab : { ...tab, resource: next };
      }),
    };
  return {
    ...node,
    first: remapNode(node.first, remap),
    last: remapNode(node.last, remap),
  };
};

type ComponentContext = Omit<Import.FileIngesterContext, "name">;

const ingestComponents = async (
  files: Import.File[],
  fileIngesters: Import.FileIngesters,
  ctx: ComponentContext,
): Promise<Map<string, ontology.ID>> => {
  const remap = new Map<string, ontology.ID>();
  for (const file of files) {
    const { data } = file;
    if (typeof data !== "object" || data == null || !("type" in data)) continue;
    if (typeof data.type !== "string") continue;
    const ingestFile = fileIngesters[data.type];
    if (ingestFile == null) continue;
    const id = await ingestFile(data, {
      ...ctx,
      name: Import.trimFileName(file.name),
    });
    if (id != null && "key" in data && typeof data.key === "string")
      remap.set(data.key, id);
  }
  return remap;
};

const ingestLegacy = async (
  legacyData: unknown,
  files: Import.File[],
  fileIngesters: Import.FileIngesters,
  ctx: ComponentContext,
): Promise<void> => {
  const { layouts } = legacySliceZ.parse(legacyData);
  for (const [key, layout] of Object.entries(layouts)) {
    const ingestFile = fileIngesters[layout.type];
    if (ingestFile == null) continue;
    const data = files.find(
      (file) =>
        file.name === `${layout.name}.json` ||
        file.name === `${key}.json` ||
        (typeof file.data === "object" &&
          file.data != null &&
          (("key" in file.data && file.data.key === key) ||
            ("name" in file.data && file.data.name === layout.name))),
    )?.data;
    if (data == null) throw new Error(`Data for ${key} not found`);
    await ingestFile(data, { ...ctx, name: layout.name });
  }
  // TODO(SY-4370): legacy exports carried a mosaic tiling for these layouts;
  // reconstructing it as panel documents is dropped, so a legacy import only
  // recreates the visualization documents.
};

export const ingest: Import.DirectoryIngester = async (
  name,
  files,
  { client, fileIngesters, store, fluxStore },
) => {
  if (!Access.updateGranted({ id: project.TYPE_ONTOLOGY_ID, store: fluxStore, client }))
    throw new Error("You do not have permission to import projects");
  if (client == null) throw new DisconnectedError();
  const panelsFile = files.find((file) => file.name === PANELS_FILE_NAME);
  const legacyFile = files.find((file) => file.name === LAYOUT_FILE_NAME);
  if (panelsFile == null && legacyFile == null)
    throw new Error(`${PANELS_FILE_NAME} not found`);
  const projectKey = uuid.create();
  // Create the project first so imported components can be parented to it; its
  // panels are created below once the components' real keys are known.
  await client.projects.create({ key: projectKey, name, layout: {} });
  const ctx: ComponentContext = {
    openTab: noopOpenTab,
    store: fluxStore,
    client,
    projectKey,
  };
  if (panelsFile != null) {
    const panels = panel.panelZ.array().parse(panelsFile.data);
    const remap = await ingestComponents(
      files.filter((file) => file.name !== PANELS_FILE_NAME),
      fileIngesters,
      ctx,
    );
    if (panels.length > 0)
      await client.panels.create(
        panels.map((p) => ({
          ...p,
          key: uuid.create(),
          root: remapNode(p.root, remap),
          parent: project.ontologyID(projectKey),
        })),
      );
  } else if (legacyFile != null)
    await ingestLegacy(legacyFile.data, files, fileIngesters, ctx);
  store.dispatch(Session.Project.select(projectKey));
};

export interface IngestContext {
  handleError: Status.ErrorHandler;
  client: Synnax | null;
  fileIngesters: Import.FileIngesters;
  openTab: Panel.OpenTab;
  store: Store;
  fluxStore: Pluto.FluxStore;
}

export const import_ = ({
  handleError,
  client,
  fileIngesters,
  openTab,
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
      openTab,
      store,
      fluxStore,
    });
  }, `Failed to import ${name}`);
};
