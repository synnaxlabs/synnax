// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status, Synnax } from "@synnaxlabs/pluto";
import { type DialogFilter, save } from "@tauri-apps/plugin-dialog";
import { writeTextFile } from "@tauri-apps/plugin-fs";
import { useStore } from "react-redux";

import { type Extractor } from "@/export/extractor";
import { type RootState } from "@/store";

const FILTERS: DialogFilter[] = [{ name: "JSON", extensions: ["json"] }];

export const use = (
  extract: Extractor,
  type = "visualization",
): ((key: string) => Promise<void>) => {
  const client = Synnax.use();
  const store = useStore<RootState>();
  const handleException = Status.useExceptionHandler();
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
      handleException(e, `Failed to export ${name ?? type}`);
    }
  };
};
