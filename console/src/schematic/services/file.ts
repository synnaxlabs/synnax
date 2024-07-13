// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Layout } from "@/layout";
import { Schematic } from "@/schematic";
import { migrateState, STATES_Z } from "@/schematic/migrations";

export const FileHandler: Layout.FileHandler = async ({
  nodeKey,
  file,
  placer,
  name,
}): Promise<boolean> => {
  console.log("STarting FileHandler");
  const z = STATES_Z.find((stateZ) => {
    return stateZ.safeParse(file).success;
  });
  if (z == null) return false;
  const state = migrateState(z.parse(file));
  placer(
    Schematic.create({
      ...state,
      name,
    }),
  );
  console.log(`Opening file ${name}.json with key ${nodeKey}`);
  return true;
};
