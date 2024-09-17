// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { NotFoundError, UnexpectedError } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { type UnknownRecord } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type DialogFilter, open, save } from "@tauri-apps/plugin-dialog";
import { readFile, writeFile } from "@tauri-apps/plugin-fs";
import { useDispatch, useStore } from "react-redux";

import { Confirm } from "@/confirm";
import { Layout } from "@/layout";
import { create } from "@/schematic/Schematic";
import { select } from "@/schematic/selectors";
import { parser, remove, type State, type StateWithName } from "@/schematic/slice";
import { RootState } from "@/store";
import { Workspace } from "@/workspace";

export const fileHandler: Layout.FileHandler = async ({
  file,
  placer,
  tab,
  dispatch,
  client,
  workspaceKey,
  confirm,
  name: fileName,
  store,
}): Promise<boolean> => {
  const state = parser(file);
  if (state == null) return false;
  const key = state.key;
  let name = file?.name;
  if (typeof name !== "string" || name.length === 0)
    name = fileName.split(".").slice(0, -1).join(".");
  if (name.length === 0) name = "New Schematic";

  const existingState = select(store.getState(), key);
  const existingName = Layout.select(store.getState(), key)?.name;

  const creator = create({
    ...state,
    tab,
    name,
  });

  if (existingState != null) {
    if (
      !(await confirm({
        message:
          `${fileName} already exists` +
          (existingName != null ? ` as ${existingName}` : ""),
        description: "Would you like to replace the existing schematic?",
        cancel: { label: "Cancel" },
        confirm: { label: "Replace", variant: "error" },
      }))
    )
      return true;
    dispatch(Layout.remove({ keys: [key] }));
    dispatch(remove({ keys: [key] }));
  }
  placer(creator);
  if (client == null) return true;

  // Logic for changing the schematic in the cluster
  try {
    await client.workspaces.schematic.retrieve(key);
    await client.workspaces.schematic.setData(key, state as unknown as UnknownRecord);
    await client.workspaces.schematic.rename(key, name);
  } catch (e) {
    if (!NotFoundError.matches(e)) throw e;
    if (workspaceKey != null)
      await client.workspaces.schematic.create(workspaceKey, {
        ...state,
        data: state as unknown as UnknownRecord,
        name,
        snapshot: state.snapshot,
        key,
      });
  }
  return true;
};

const filters: DialogFilter[] = [{ name: "JSON", extensions: ["json"] }];

export const useExport = (name: string = "schematic"): ((key: string) => void) => {
  const client = Synnax.use();
  const addStatus = Status.useAggregator();
  const store = useStore<RootState>();

  return useMutation<void, Error, string>({
    mutationFn: async (key) => {
      const storeState = store.getState();
      let state = select(storeState, key);
      let name = Layout.select(storeState, key)?.name;
      if (state == null) {
        if (client == null) throw new Error("Cannot reach cluster");
        const schematic = await client.workspaces.schematic.retrieve(key);
        state = {
          ...(schematic.data as unknown as State),
          snapshot: schematic.snapshot,
          key: schematic.key,
        };
        name = schematic.name;
      }
      if (name == null)
        throw new UnexpectedError("Cannot find name of schematic to export");
      const savePath = await save({
        title: `Export ${name}`,
        defaultPath: `${name}.json`,
        filters,
      });
      if (savePath == null) return;
      const schematicData: StateWithName = { ...state, name };
      await writeFile(
        savePath,
        new TextEncoder().encode(JSON.stringify(schematicData)),
      );
    },
    onError: (err) =>
      addStatus({
        variant: "error",
        message: `Failed to export ${name}`,
        description: err.message,
      }),
  }).mutate;
};

export const useImport = (workspaceKey?: string): (() => void) => {
  const placeLayout = Layout.usePlacer();
  const addStatus = Status.useAggregator();
  const store = useStore<RootState>();
  const confirm = Confirm.useModal();
  const client = Synnax.use();
  const dispatch = useDispatch();
  const activeKey = Workspace.useSelectActiveKey();
  if (workspaceKey != null && activeKey !== workspaceKey)
    dispatch(Workspace.setActive(workspaceKey));

  return useMutation<void, Error>({
    mutationFn: async () => {
      const fileResponses = await open({
        title: "Import schematic",
        filters,
        multiple: true,
        directory: false,
      });
      if (fileResponses == null) return;
      for (const path of fileResponses) {
        const rawData = await readFile(path);
        const fileName = path.split("/").pop();
        if (fileName == null) throw new UnexpectedError("File name is null");
        const file = JSON.parse(new TextDecoder().decode(rawData));
        if (
          !(await fileHandler({
            file,
            placer: placeLayout,
            name: fileName,
            store,
            confirm,
            client,
            workspaceKey,
            dispatch,
          }))
        )
          throw new Error(`${fileName} is not a valid schematic`);
      }
    },
    onError: (err) =>
      addStatus({
        variant: "error",
        message: `Failed to import schematic`,
        description: err.message,
      }),
  }).mutate;
};
