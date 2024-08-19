// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { type UnknownRecord } from "@synnaxlabs/x";

import { Layout } from "@/layout";
import { parser } from "@/schematic/migrations";
import { create } from "@/schematic/Schematic";
import { select } from "@/schematic/selectors";
import { remove } from "@/schematic/slice";

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
  const newState = parser(file);
  if (newState == null) return false;
  const key = newState.key;
  const existingState = select(store.getState(), key);
  const existingName = Layout.select(store.getState(), key)?.name;
  const potentialFileName = file?.name;
  let name = typeof potentialFileName === "string" ? potentialFileName : undefined;
  if (name == null) name = fileName.split(".").slice(0, -1).join(".");
  if (name.length === 0) name = "New Schematic";

  const creator = create({
    ...newState,
    tab,
    name: fileName,
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
  } else if (client != null) {
    try {
      const existing = await client.workspaces.schematic.retrieve(key);
      if (
        !(await confirm({
          message: `${fileName} already exists in the cluster as ${existing.name}`,
          description: "Would you like to replace the existing schematic?",
          cancel: { label: "Cancel" },
          confirm: { label: "Replace", variant: "error" },
        }))
      )
        return true;
    } catch (e) {
      if (!NotFoundError.matches(e)) throw e;
    }
  }
  dispatch(Layout.remove({ keys: [key] }));
  dispatch(remove({ keys: [key] }));

  if (client == null) {
    placer(creator);
    return true;
  }

  // Logic for changing the schematic in the cluster
  try {
    await client.workspaces.schematic.retrieve(key);
    await client.workspaces.schematic.setData(
      key,
      newState as unknown as UnknownRecord,
    );
    await client.workspaces.schematic.rename(key, name);
  } catch (e) {
    if (!NotFoundError.matches(e)) throw e;
    if (workspaceKey != null)
      await client.workspaces.schematic.create(workspaceKey, {
        data: newState as unknown as UnknownRecord,
        name,
        snapshot: newState.snapshot,
        key,
      });
  }
  placer(creator);
  return true;
};
