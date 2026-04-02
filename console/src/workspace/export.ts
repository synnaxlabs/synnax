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
import { join } from "@tauri-apps/api/path";
import { open } from "@tauri-apps/plugin-dialog";
import { exists, mkdir, writeTextFile } from "@tauri-apps/plugin-fs";
import { strToU8, zipSync } from "fflate";
import { useStore } from "react-redux";

import { type Export } from "@/export";
import { useExtractors } from "@/export/ExtractorsProvider";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { Runtime } from "@/runtime";
import { type RootAction, type RootState, type RootStore } from "@/store";
import { purgeExcludedLayouts } from "@/workspace/purgeExcludedLayouts";
import { selectActive } from "@/workspace/selectors";

const removeDirectory = (name: string): string => name.replace(/[/\\]/g, "_");

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
  let name: string = "workspace";
  handleError(async () => {
    const storeState = store.getState();
    const active = selectActive(storeState);
    let toExport: Layout.SliceState;
    if (active?.key === key || key == null) {
      const file = Layout.selectSliceState(storeState);
      toExport = purgeExcludedLayouts(file);
      if (active?.key != null) name = active.name;
    } else {
      if (client == null) throw new DisconnectedError();
      const ws = await client.workspaces.retrieve(key);
      toExport = ws.layout as Layout.SliceState;
      name = ws.name;
    }

    // In Tauri mode, prompt for a destination directory before doing extraction
    // work so the user can cancel without wasting effort.
    let directory: string | undefined;
    if (Runtime.ENGINE === "tauri") {
      const parentDir = await open({
        directory: true,
        title: `Select a location to export ${name}`,
        recursive: true,
      });
      if (parentDir == null) return;
      directory = await join(parentDir, removeDirectory(name));
      if (
        (await exists(directory)) &&
        !(await confirm({
          message: `A file or directory already exists at ${directory}`,
          description: "Replacing will cause the old data to be deleted.",
          cancel: { label: "Cancel" },
          confirm: { label: "Replace", variant: "error" },
        }))
      )
        return;
      await mkdir(directory, { recursive: true });
    }

    const namesSet = new Set<string>();
    Object.values(toExport.layouts).forEach((layout) => {
      const deduplicatedName = strings.deduplicateFileName(layout.name, namesSet);
      layout.name = removeDirectory(deduplicatedName);
      namesSet.add(layout.name);
    });

    const fileInfos: Export.File[] = [];
    await Promise.all(
      Object.values(toExport.layouts).map(async ({ type, key, name }) => {
        const extractor = extractors[type];
        if (extractor == null) return;
        const { data } = await extractor(key, { store, client });
        fileInfos.push({ data, name: `${name}.json` });
      }),
    );

    if (Runtime.ENGINE === "tauri") {
      await writeTextFile(
        await join(directory!, LAYOUT_FILE_NAME),
        JSON.stringify(toExport),
      );
      await Promise.all(
        fileInfos.map(async ({ data, name }) => {
          await writeTextFile(await join(directory!, name), data);
        }),
      );
      addStatus({ variant: "success", message: `Exported ${name} to ${directory}` });
    } else {
      // Browsers cannot write to a user-selected directory, so we bundle all
      // files into a single .zip archive for download.
      const zipFiles: Record<string, Uint8Array> = {
        [LAYOUT_FILE_NAME]: strToU8(JSON.stringify(toExport)),
      };
      fileInfos.forEach(({ data, name }) => {
        zipFiles[name] = strToU8(data);
      });
      const zipped = zipSync(zipFiles);
      Runtime.downloadFromBrowser(
        new Blob([new Uint8Array(zipped)], { type: "application/zip" }),
        `${removeDirectory(name)}.zip`,
      );
      addStatus({ variant: "success", message: `Exported ${name}` });
    }
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
