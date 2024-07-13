// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Layout } from "@/layout";
import { migrateState, STATES_Z } from "@/schematic/migrations";
import { create } from "@/schematic/Schematic";

export const FileHandler: Layout.FileHandler = async ({
  // nodeKey,
  file,
  placer,
  name,
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
  const foo = placer(creator);
  console.log(foo);
  return true;
};
