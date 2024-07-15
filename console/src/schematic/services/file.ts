// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { deep, errors, type UnknownRecord } from "@synnaxlabs/x";

import { Layout } from "@/layout";
import { moveMosaicTab } from "@/layout/slice";
import { parser } from "@/schematic/migrations";
import { create } from "@/schematic/Schematic";
import { select } from "@/schematic/selectors";
import { remove } from "@/schematic/slice";

export const FileHandler: Layout.FileHandler = async ({
  mosaicKey,
  file,
  placer,
  loc,
  name,
  client,
  workspaceKey,
  confirm,
  dispatch,
  store,
}): Promise<boolean> => {
  const newState = parser(file);
  if (newState == null) return false;
  const creator = create({
    ...newState,
    name,
  });
  const key = newState.key;
  const existingState = select(store.getState(), key);
  if (existingState != null) {
    if (deep.equal(existingState, newState)) throw Error(`${name} already exists.`);
    if (
      !(await confirm({
        message: `${name} already exists`,
        description: "Would you like to replace the existing schematic?",
        cancel: { label: "Cancel" },
        confirm: { label: "Replace", variant: "error" },
      }))
    )
      throw errors.CANCELED;
    Layout.remove({ keys: [key] });
    remove({ keys: [key] });
  }
  if (client != null) {
    try {
      await client.workspaces.schematic.retrieve(key);
      await client.workspaces.schematic.setData(
        key,
        newState as unknown as UnknownRecord,
      );
    } catch (e) {
      if (!NotFoundError.matches(e)) throw e;
      if (workspaceKey == null) throw Error("Workspace key is required.");
      await client.workspaces.schematic.create(workspaceKey, {
        name,
        data: newState as unknown as UnknownRecord,
        ...newState,
      });
    }
  }
  const windowKey = placer(creator).windowKey;
  dispatch(
    moveMosaicTab({
      key: mosaicKey,
      windowKey,
      tabKey: newState.key,
      loc,
    }),
  );
  return true;
};
