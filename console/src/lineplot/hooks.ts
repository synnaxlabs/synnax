// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";
import { Status, Synnax, Triggers, useSyncedRef } from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";
import { open, save } from "@tauri-apps/plugin-dialog";
import { readFile, writeFile } from "@tauri-apps/plugin-fs";
import { useCallback } from "react";
import { useDispatch, useStore } from "react-redux";

import { Confirm } from "@/confirm";
import { Layout } from "@/layout";
import { select, useSelectControlState } from "@/lineplot/selectors";
import { fileHandler } from "@/lineplot/services/file";
import { setControlState, type State, type StateWithName } from "@/lineplot/slice";
import { type RootState } from "@/store";
import { Workspace } from "@/workspace";

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
      const storeState = store.getState();
      let state = select(storeState, key);
      let name = Layout.select(storeState, key)?.name;
      if (state == null) {
        if (client == null) throw new Error("Client is not available");
        const linePlot = await client.workspaces.linePlot.retrieve(key);
        state = {
          ...(linePlot.data as unknown as State),
          key: linePlot.key,
        };
        name = linePlot.name;
      }
      if (name == null)
        throw new UnexpectedError(
          `Line plot with key ${key} is missing in store state`,
        );
      const savePath = await save({
        defaultPath: `${name}.json`,
        filters: [{ name: "JSON", extensions: ["json"] }],
      });
      if (savePath == null) return;
      const linePlotData: StateWithName = { ...state, name };
      await writeFile(savePath, new TextEncoder().encode(JSON.stringify(linePlotData)));
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

export const useImport = (workspaceKey?: string): (() => void) => {
  const placer = Layout.usePlacer();
  const addStatus = Status.useAggregator();
  const store = useStore<RootState>();
  const confirm = Confirm.useModal();
  const client = Synnax.use();
  const dispatch = useDispatch();
  const activeKey = Workspace.useSelectActiveKey();
  if (workspaceKey != null && activeKey !== workspaceKey)
    dispatch(Workspace.setActive(workspaceKey));

  let name = "line plot";

  return useMutation<void, Error>({
    mutationFn: async () => {
      const fileResponse = await open({
        directory: false,
        multiple: false,
        title: `Import schematic`,
        extenstions: ["json"],
      });
      if (fileResponse == null) return;
      const rawData = await readFile(fileResponse.path);
      const fileName = fileResponse.path.split("/").pop();
      if (fileName == null) throw new UnexpectedError("File name is null");
      name = fileName;
      const file = JSON.parse(new TextDecoder().decode(rawData));
      if (
        !(await fileHandler({
          file,
          placer,
          name: fileName,
          store,
          confirm,
          client,
          workspaceKey,
          dispatch,
        }))
      )
        throw new Error(`${fileName} is not a valid schematic`);
    },
    onError: (err) => {
      addStatus({
        variant: "error",
        message: `Failed to import ${name}`,
        description: err.message,
      });
    },
  }).mutate;
};
