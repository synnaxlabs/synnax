// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, type Synnax as Client } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { strings } from "@synnaxlabs/x";
import { useStore } from "react-redux";

import { Export } from "@/export";
import { useExtractors } from "@/export/ExtractorsProvider";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { purgeExcludedLayouts } from "@/project/purgeExcludedLayouts";
import { selectActive } from "@/project/selectors";
import { Runtime } from "@/runtime";
import { type RootAction, type RootState, type RootStore } from "@/store";

export interface ExportContext {
  client: Client | null;
  store: RootStore;
  confirm: Modals.PromptConfirm;
  handleError: Status.ErrorHandler;
  extractors: Export.Extractors;
  addStatus: Status.Adder;
}

export const export_ = (
  key: string | null,
  { client, store, confirm, handleError, extractors, addStatus }: ExportContext,
): void => {
  let name: string = "project"; // default name for error message
  handleError(async () => {
    const storeState = store.getState();
    const active = selectActive(storeState);
    let toExport: Layout.SliceState;
    if (active.key === key || key == null) {
      const file = Layout.selectSliceState(storeState);
      toExport = purgeExcludedLayouts(file);
      name = active.name;
    } else {
      if (client == null) throw new DisconnectedError();
      const proj = await client.projects.retrieve(key);
      toExport = proj.layout as Layout.SliceState;
      name = proj.name;
    }
    const directory = await Runtime.pickWritableDirectory({
      title: `Select a location to export ${name}`,
      subdirectory: Export.sanitizeFileName(name),
    });
    if (directory == null) return;
    if (
      directory.preExisted &&
      !(await confirm({
        message: `A file or directory already exists at ${directory.displayPath}`,
        description: "Replacing will cause the old data to be deleted.",
        cancel: { label: "Cancel" },
        confirm: { label: "Replace", variant: "error" },
      }))
    )
      return;
    const namesSet = new Set<string>();
    Object.values(toExport.layouts).forEach((layout) => {
      const deduplicatedName = strings.deduplicateFileName(layout.name, namesSet);
      layout.name = Export.sanitizeFileName(deduplicatedName);
      namesSet.add(layout.name);
    });
    await directory.writeText(LAYOUT_FILE_NAME, JSON.stringify(toExport));
    const fileInfos: Export.File[] = [];
    await Promise.all(
      Object.values(toExport.layouts).map(async ({ type, key }) => {
        const extractor = extractors[type];
        if (extractor == null) return;
        const { data } = await extractor(key, { store, client });
        fileInfos.push({ data, name: `${toExport.layouts[key].name}.json` });
      }),
    );
    await Promise.all(
      fileInfos.map(({ data, name }) => directory.writeText(name, data)),
    );
    addStatus({
      variant: "success",
      message: `Exported ${name} to ${directory.displayPath}`,
    });
  }, `Failed to export ${name}`);
};

export const LAYOUT_FILE_NAME = "LAYOUT.json";

export const useExport = (): ((key: string | null) => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const store = useStore<RootState, RootAction>();
  const confirm = Modals.useConfirm();
  const extractors = useExtractors();
  return (key: string | null) =>
    export_(key, { client, store, confirm, handleError, extractors, addStatus });
};

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
  store.dispatch(Project.activeate({ key: projectKey, name }));

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
