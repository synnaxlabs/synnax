// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { NotFoundError, workspace as cWorkspace } from "@synnaxlabs/client";
import { Status, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";
import { save } from "@tauri-apps/plugin-dialog";
import { writeFile } from "@tauri-apps/plugin-fs";
import { useStore } from "react-redux";

import { Layout } from "@/layout";
import { type RootState } from "@/store";
import { select } from "@/workspace/selectors";
import { add, remove } from "@/workspace/slice";

export const fileHandler: Layout.FileHandler = async ({
  file,
  dispatch,
  client,
  confirm,
  name,
  store,
}): Promise<boolean> => {
  const workspaceZ = cWorkspace.workspaceZ;
  if (!workspaceZ.safeParse(file).success) return false;
  const workspace = workspaceZ.parse(file);
  const key = workspace.key;

  const existingWorkspace = select(store.getState(), key);
  if (existingWorkspace != null) {
    if (
      !(await confirm({
        message: `${name} already exists as ${existingWorkspace.name}`,
        description: "Would you like to replace the existing workspace?",
        cancel: { label: "Cancel" },
        confirm: { label: "Replace", variant: "error" },
      }))
    )
      return true;
    dispatch(remove({ keys: [key] }));
  }
  dispatch(add({ workspaces: [workspace] }));
  dispatch(
    Layout.setWorkspace({
      slice: workspace.layout as unknown as Layout.SliceState,
      keepNav: false,
    }),
  );

  if (client == null) return true;

  // Logic for changing the workspace in the cluster
  try {
    await client.workspaces.retrieve(key);
    await client.workspaces.rename(key, workspace.name);
    await client.workspaces.setLayout(key, workspace.layout);
  } catch (e) {
    if (!NotFoundError.matches(e)) throw e;
    await client.workspaces.create(workspace);
  }
  return true;
};

export const useExport = (name: string = "workspace"): ((key: string) => void) => {
  const client = PSynnax.use();
  const addStatus = Status.useAggregator();
  const store = useStore<RootState>();

  return useMutation<void, Error, string>({
    mutationFn: async (key) => {
      let workspace = select(store.getState(), key);
      if (workspace == null) {
        if (client == null) throw new Error("Client is not available");
        workspace = await client.workspaces.retrieve(key);
      }
      const savePath = await save({
        title: `Export ${name}`,
        defaultPath: `${workspace.name}.json`,
        filters: [{ name: "JSON", extensions: ["json"] }],
      });
      if (savePath == null) return;
      await writeFile(savePath, new TextEncoder().encode(JSON.stringify(workspace)));
    },
    onError: (err) =>
      addStatus({
        variant: "error",
        message: `Failed to export ${name}`,
        description: err.message,
      }),
  }).mutate;
};
