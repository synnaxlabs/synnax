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
  name,
  store,
}): Promise<boolean> => {
  const newState = parser(file);
  if (newState == null) return false;
  const creator = create({
    ...newState,
    tab,
  });
  const key = newState.key;
  const existingState = select(store.getState(), key);
  if (existingState != null) {
    if (
      !(await confirm({
        message: `${name} already exists as ${existingState.name}`,
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
    await client.workspaces.schematic.setData(
      key,
      newState as unknown as UnknownRecord,
    );
  } catch (e) {
    if (!NotFoundError.matches(e)) throw e;
    if (workspaceKey != null)
      await client.workspaces.schematic.create(workspaceKey, {
        data: newState as unknown as UnknownRecord,
        ...newState,
      });
  }
  return true;
};
