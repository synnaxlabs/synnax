// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status, Synnax } from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";
import { save } from "@tauri-apps/plugin-dialog";
import { writeFile } from "@tauri-apps/plugin-fs";
import { useStore } from "react-redux";

import { select } from "@/schematic/selectors";
import { type State } from "@/schematic/slice";
import { RootState } from "@/store";

export const useDownload = (name: string = "schematic"): ((key: string) => void) => {
  const client = Synnax.use();
  const addStatus = Status.useAggregator();
  const store = useStore<RootState>();

  return useMutation<void, Error, string>({
    mutationFn: async (key) => {
      let state = select(store.getState(), key);
      if (state == null) {
        if (client == null) throw new Error("Client is not available");
        const schematic = await client.workspaces.schematic.retrieve(key);
        state = schematic.data as unknown as State;
      }
      const savePath = await save({
        defaultPath: `${name}.json`,
        filters: [{ name: "JSON", extensions: ["json"] }],
      });
      if (savePath == null) return;
      await writeFile(savePath, new TextEncoder().encode(JSON.stringify(state)));
    },
    onError: (err) => {
      addStatus({
        variant: "error",
        message: `Failed to download ${name}`,
        description: err.message,
      });
    },
  }).mutate;
};
