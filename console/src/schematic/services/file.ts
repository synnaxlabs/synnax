// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { deep, type UnknownRecord } from "@synnaxlabs/x";

import { Layout } from "@/layout";
import { moveMosaicTab } from "@/layout/slice";
import { migrateState, STATES_Z } from "@/schematic/migrations";
import { create } from "@/schematic/Schematic";

export const FileHandler: Layout.FileHandler = async ({
  mosaicKey,
  file,
  placer,
  location,
  name,
  client,
  workspaceKey,
  dispatch,
}): Promise<boolean> => {
  const z = STATES_Z.find((stateZ) => {
    return stateZ.safeParse(file).success;
  });
  if (z == null) return false;
  const state = migrateState(z.parse(file));
  const creator = create({
    ...state,
    name,
  });
  if (client == null) {
    placer(creator);
    return true;
  }
  try {
    const schematic = await client.workspaces.schematic.retrieve(state.key);
    await client.workspaces.schematic.setData(
      schematic.key,
      state as unknown as UnknownRecord,
    );
  } catch (e) {
    if (!NotFoundError.matches(e)) throw e;
    if (workspaceKey == null) {
      const foo = placer(creator);
      console.log("foo", foo);
      dispatch(
        moveMosaicTab({
          key: mosaicKey,
          windowKey: foo.windowKey,
          tabKey: foo.key,
          loc: location,
        }),
      );
      return true;
    }
    await client.workspaces.schematic.create(workspaceKey, {
      name,
      data: deep.copy(state) as unknown as UnknownRecord,
      ...state,
    });
  }
  const foo = placer(creator);
  console.log("foo", foo);
  dispatch(
    moveMosaicTab({
      key: mosaicKey,
      windowKey: foo.windowKey,
      tabKey: state.key,
      loc: location,
    }),
  );
  return true;
};
