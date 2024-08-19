// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status, Synnax, Triggers, useSyncedRef } from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";
import { save } from "@tauri-apps/plugin-dialog";
import { writeFile } from "@tauri-apps/plugin-fs";
import { useCallback } from "react";
import { useDispatch, useStore } from "react-redux";

import { select, useSelectControlState } from "@/lineplot/selectors";
import { setControlState, type State } from "@/lineplot/slice";
import { type RootState } from "@/store";

export type Config = Triggers.ModeConfig<"toggle" | "hold">;

export const useTriggerHold = (triggers: Config): void => {
  const { hold } = useSelectControlState();
  const ref = useSyncedRef(hold);
  const triggersRef = useSyncedRef(triggers);
  const d = useDispatch();
  const flat = Triggers.useFlattenedMemoConfig(triggers);
  Triggers.use({
    triggers: flat,
    loose: true,
    callback: useCallback(
      (e: Triggers.UseEvent) => {
        const mode = Triggers.determineMode(triggersRef.current, e.triggers);
        if (mode === "hold") {
          if (e.stage === "start") d(setControlState({ state: { hold: true } }));
          else if (e.stage === "end") d(setControlState({ state: { hold: false } }));
          return;
        }
        if (e.stage !== "start") return;
        d(setControlState({ state: { hold: !ref.current } }));
      },
      [d],
    ),
  });
};

export const useExport = (name: string = "line plot"): ((key: string) => void) => {
  const client = Synnax.use();
  const addStatus = Status.useAggregator();
  const store = useStore<RootState>();

  return useMutation<void, Error, string>({
    mutationFn: async (key) => {
      let state = select(store.getState(), key);
      if (state == null) {
        if (client == null) throw new Error("Client is not available");
        const linePlot = await client.workspaces.linePlot.retrieve(key);
        state = linePlot.data as unknown as State;
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
        message: `Failed to export ${name}`,
        description: err.message,
      });
    },
  }).mutate;
};

export const useImport = () => () => console.log("Importing line plot");
