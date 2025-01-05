// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { Status, Synnax } from "@synnaxlabs/pluto";
import { join } from "@tauri-apps/api/path";
import { open } from "@tauri-apps/plugin-dialog";
import { mkdir, writeTextFile } from "@tauri-apps/plugin-fs";
import { sep } from "path";
import { useStore } from "react-redux";

import { EXTRACTORS } from "@/extractors";
import { type Export } from "@/export";
import { Layout } from "@/layout";
import { type RootState } from "@/store";
import { convertLayout } from "@/workspace/convertLayout";
import { select, selectActiveKey } from "@/workspace/selectors";

const removeDirectory = (name: string): string => name.split(sep).join("_");

export const useExport = (): ((key: string) => Promise<void>) => {
  const client = Synnax.use();
  const addStatus = Status.useAggregator();
  const store = useStore<RootState>();
  return async (key: string) => {
    let name: string = "workspace"; // default name for error message
    try {
      const storeState = store.getState();
      const activeKey = selectActiveKey(storeState);
      let toExport: Layout.SliceState;
      if (activeKey === key) {
        // active workspace is the current workspace
        const file = Layout.selectSliceState(storeState);
        toExport = convertLayout(file);
        name = select(storeState, key)?.name ?? "Workspace";
      } else {
        const existingWorkspace = select(storeState, key);
        if (existingWorkspace != null) {
          // key exists in store
          toExport = existingWorkspace.layout as Layout.SliceState;
          name = existingWorkspace.name;
        } else {
          // fetch workspace from cluster
          if (client == null) throw new Error("Cannot reach cluster");
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
      const directory = await join(parentDir, name);
      await mkdir(directory, { recursive: true });
      await writeTextFile(
        await join(directory, `${removeDirectory(name)}.json`),
        JSON.stringify(toExport),
      );
      const fileInfos: Export.ExtractorReturn[] = [];
      await Promise.all(
        Object.values(toExport.layouts).map(async ({ type, key }) => {
          const extractor = EXTRACTORS[type];
          if (extractor == null) return;
          fileInfos.push(await extractor(key, { store, client }));
        }),
      );
      const names = new Set<string>();
      for (const fileInfo of fileInfos) {
        // make sure we don't overwrite files with the same name, so unique files for
        // each layout will exist. Also remove directory separators from the name.
        fileInfo.name = removeDirectory(fileInfo.name);
        while (names.has(fileInfo.name)) fileInfo.name += "_";
        names.add(fileInfo.name);
      }
      await Promise.all(
        fileInfos.map(async ({ file, name }) => {
          await writeTextFile(await join(directory, `${name}.json`), file);
        }),
      );
    } catch (e) {
      if (!(e instanceof Error)) throw e;
      addStatus({
        variant: "error",
        message: `Failed to export ${name}`,
        description: e.message,
      });
    }
  };
};
