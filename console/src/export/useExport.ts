// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { Status, Synnax } from "@synnaxlabs/pluto";
import { type DialogFilter, save } from "@tauri-apps/plugin-dialog";
import { writeTextFile } from "@tauri-apps/plugin-fs";
import { useStore } from "react-redux";

import { type FileExtractor } from "@/export/FileExtractor";
import { type RootState } from "@/store";

const FILTERS: DialogFilter[] = [{ name: "JSON", extensions: ["json"] }];

export const useExport = (extract: FileExtractor): ((key: string) => Promise<void>) => {
  const client = Synnax.use();
  const store = useStore<RootState>();
  const addStatus = Status.useAggregator();
  return async (key: string) => {
    let name;
    try {
      const extractorReturn = await extract(key, { store, client });
      name = extractorReturn.name;
      const savePath = await save({
        title: `Export ${name}`,
        defaultPath: `${name}.json`,
        filters: FILTERS,
      });
      if (savePath == null) return;
      await writeTextFile(savePath, extractorReturn.data);
    } catch (e) {
      if (!(e instanceof Error)) throw e;
      addStatus({
        variant: "error",
        message: `Failed to export ${name ?? "layout"}`,
        description: e.message,
      });
    }
  };
};
