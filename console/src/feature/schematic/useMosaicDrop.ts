// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Layout } from "@/platform/layout";
import { type Mosaic } from "@/platform/mosaic";
import { Schematic } from "@/platform/schematic";

export const useMosaicDrop = (): Mosaic.DropHandler => {
  const client = Synnax.use();
  const placeLayout = Layout.usePlacer();
  const handleError = Status.useErrorHandler();
  return useCallback(
    ({ id: { key }, nodeKey, location }) => {
      if (client == null) return;
      handleError(async () => {
        const schematic = await client.schematics.retrieve({ key });
        placeLayout(
          Schematic.create({
            key: schematic.key,
            name: schematic.name,
            tab: { mosaicKey: nodeKey, location },
          }),
        );
      }, "Failed to load schematic");
    },
    [client, placeLayout, handleError],
  );
};
