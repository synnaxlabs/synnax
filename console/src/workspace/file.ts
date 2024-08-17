// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { NotFoundError, workspace as cWorkspace } from "@synnaxlabs/client";

import { Layout } from "@/layout";
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
