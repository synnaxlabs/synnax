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
import { type DialogFilter, open } from "@tauri-apps/plugin-dialog";
import { readFile } from "@tauri-apps/plugin-fs";
import { useDispatch, useStore } from "react-redux";

import { Confirm } from "@/confirm";
import { type File } from "@/file";
import { Layout } from "@/layout";
import { create } from "@/schematic/Schematic";
import { select, selectHasPermission } from "@/schematic/selectors";
import { parser, remove } from "@/schematic/slice";
import { type RootState } from "@/store";
import { Workspace } from "@/workspace";

export const fileHandler: File.FileHandler = async ({
  file,
  place,
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
  const canCreate = selectHasPermission(store.getState());
  if (!canCreate)
    throw new Error(
      "You do not have permission to create a schematic. Please contact an admin to change your permissions.",
    );
  const key = state.key;
  let name = "";
  if (typeof file?.name === "string") name = file.name;
  if (name.length === 0) name = fileName.split(".").slice(0, -1).join(".");
  if (name.length === 0) name = "New Schematic";

  const existingState = select(store.getState(), key);
  const existingName = Layout.select(store.getState(), key)?.name;
  const creator = create({ ...state, tab, name });

  if (existingState != null) {
    if (
      !(await confirm({
        message: `${fileName} already exists${
          existingName != null ? ` as ${existingName}` : ""
        }`,
        description: "Would you like to replace the existing schematic?",
        cancel: { label: "Cancel" },
        confirm: { label: "Replace", variant: "error" },
      }))
    )
      return true;
    dispatch(Layout.remove({ keys: [key] }));
    dispatch(remove({ keys: [key] }));
  }
  place(creator);
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

interface ImportPlotProps extends Omit<File.FileHandlerProps, "file" | "name"> {
  workspaceKey?: string;
  activeWorkspaceKey: string | null;
}

export const importSchematic = async ({
  workspaceKey,
  activeWorkspaceKey,
  store,
  place,
  confirm,
  client,
  dispatch,
}: ImportPlotProps): Promise<void> => {
  const paths = await open({
    title: "Import schematic",
    filters,
    multiple: true,
    directory: false,
  });
  if (paths == null) return;
  if (workspaceKey != null && activeWorkspaceKey !== workspaceKey) {
    let ws = Workspace.select(store.getState(), workspaceKey);
    if (ws == null) {
      if (client == null) throw new Error("Cannot reach cluster");
      ws = await client.workspaces.retrieve(workspaceKey);
    }
    dispatch(Workspace.add({ workspaces: [ws] }));
    dispatch(
      Layout.setWorkspace({
        slice: ws.layout as unknown as Layout.SliceState,
        keepNav: false,
      }),
    );
  }
  for (const path of paths) {
    const rawData = await readFile(path);
    const fileName = path.split("/").pop();
    if (fileName == null) throw new UnexpectedError("File name is null");
    const file = JSON.parse(new TextDecoder().decode(rawData));
    if (
      !(await fileHandler({
        file,
        place,
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
};

export const useImport = (workspaceKey?: string): (() => void) => {
  const placeLayout = Layout.usePlacer();
  const addStatus = Status.useAggregator();
  const store = useStore<RootState>();
  const confirm = Confirm.useModal();
  const client = Synnax.use();
  const dispatch = useDispatch();
  const activeWorkspaceKey = Workspace.useSelectActiveKey();

  return useMutation<void, Error>({
    mutationFn: async () =>
      await importSchematic({
        workspaceKey,
        activeWorkspaceKey,
        store,
        place: placeLayout,
        confirm,
        client,
        dispatch,
      }),
    onError: (err) =>
      addStatus({
        variant: "error",
        message: `Failed to import schematic`,
        description: err.message,
      }),
  }).mutate;
};
