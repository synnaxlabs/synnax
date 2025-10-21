// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, type Synnax as Client } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { join, sep } from "@tauri-apps/api/path";
import { open } from "@tauri-apps/plugin-dialog";
import { exists, mkdir, writeTextFile } from "@tauri-apps/plugin-fs";
import { useStore } from "react-redux";

import { type Export } from "@/export";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { Runtime } from "@/runtime";
import { type RootAction, type RootState, type RootStore } from "@/store";
import { purgeExcludedLayouts } from "@/workspace/purgeExcludedLayouts";
import { select, selectActiveKey } from "@/workspace/selectors";

const removeDirectory = (name: string): string => name.split(sep()).join("_");

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
  let name: string = "workspace"; // default name for error message
  handleError(async () => {
    if (Runtime.ENGINE !== "tauri")
      throw new Error("Cannot export workspaces when running Synnax in the browser.");
    const storeState = store.getState();
    const activeKey = selectActiveKey(storeState);
    let toExport: Layout.SliceState;
    if (activeKey === key || key == null) {
      const file = Layout.selectSliceState(storeState);
      toExport = purgeExcludedLayouts(file);
      if (activeKey != null) name = select(storeState, activeKey)?.name ?? "workspace";
    } else {
      const existingWorkspace = select(storeState, key);
      if (existingWorkspace != null) {
        toExport = existingWorkspace.layout as Layout.SliceState;
        name = existingWorkspace.name;
      } else {
        if (client == null) throw new DisconnectedError();
        const ws = await client.workspaces.retrieve(key);
        toExport = ws.layout as Layout.SliceState;
        name = ws.name;
      }
    }
    const parentDir = await open({
      directory: true,
      title: `Select a location to export ${name}`,
      recursive: true,
    });
    if (parentDir == null) return;
    const directory = await join(parentDir, removeDirectory(name));
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
    await writeTextFile(
      await join(directory, LAYOUT_FILE_NAME),
      JSON.stringify(toExport),
    );
    const fileInfos: Export.FileInfo[] = [];
    await Promise.all(
      Object.values(toExport.layouts).map(async ({ type, key, name }) => {
        const extractor = extractors[type];
        if (extractor == null) return;
        const fileData = (await extractor(key, { store, client })).data;
        const fileName = `${name}.json`;
        fileInfos.push({ data: fileData, name: fileName });
      }),
    );
    await Promise.all(
      fileInfos.map(async ({ data, name }) => {
        await writeTextFile(await join(directory, name), data);
      }),
    );
    addStatus({ variant: "success", message: `Exported ${name} to ${directory}` });
  }, `Failed to export ${name}`);
};

export const LAYOUT_FILE_NAME = "LAYOUT.json";

export const useExport = (extractors: Export.Extractors): ((key: string) => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const store = useStore<RootState, RootAction>();
  const confirm = Modals.useConfirm();
  return (key: string) =>
    export_(key, { client, store, confirm, handleError, extractors, addStatus });
};
