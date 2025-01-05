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
import { create } from "@/lineplot/LinePlot";
import { select } from "@/lineplot/selectors";
import { parser, remove } from "@/lineplot/slice";
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
  const linePlot = parser(file);
  if (linePlot == null) return false;
  const key = linePlot.key;
  let name = "";
  if (typeof file?.name === "string") name = file.name;
  if (name.length === 0) name = fileName.split(".").slice(0, -1).join(".");
  if (name.length === 0) name = "New Schematic";

  const creator = create({ ...linePlot, tab, name });

  const existingState = select(store.getState(), key);
  const existingName = Layout.select(store.getState(), key)?.name;

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

  // Logic for changing the line plot in the cluster
  try {
    await client.workspaces.linePlot.retrieve(key);
    await client.workspaces.linePlot.setData(key, linePlot);
    await client.workspaces.linePlot.rename(key, name);
  } catch (e) {
    if (!NotFoundError.matches(e)) throw e;
    if (workspaceKey != null)
      await client.workspaces.linePlot.create(workspaceKey, {
        ...linePlot,
        name,
        data: linePlot as unknown as UnknownRecord,
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

export const importPlot = async ({
  workspaceKey,
  activeWorkspaceKey,
  store,
  dispatch,
  confirm,
  client,
  place,
}: ImportPlotProps) => {
  const paths = await open({
    title: "Import line plot",
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
      throw new Error(`${fileName} is not a valid line plot`);
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
    mutationFn: async () => {
      await importPlot({
        workspaceKey,
        activeWorkspaceKey,
        store,
        dispatch,
        confirm,
        client,
        place: placeLayout,
      });
    },
    onError: (err) =>
      addStatus({
        variant: "error",
        message: `Failed to import line plot`,
        description: err.message,
      }),
  }).mutate;
};
