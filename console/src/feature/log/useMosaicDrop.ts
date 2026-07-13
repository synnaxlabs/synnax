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
import { Log } from "@/platform/log";
import { type Mosaic } from "@/platform/mosaic";

export const useMosaicDrop = (): Mosaic.DropHandler => {
  const client = Synnax.use();
  const placeLayout = Layout.usePlacer();
  const handleError = Status.useErrorHandler();
  return useCallback(
    ({ id: { key }, nodeKey, location }) => {
      if (client == null) return;
      handleError(async () => {
        const l = await client.logs.retrieve({ key });
        placeLayout(
          Log.create({
            key: l.key,
            name: l.name,
            location: "mosaic",
            tab: { mosaicKey: nodeKey, location },
          }),
        );
      }, "Failed to load log");
    },
    [client, placeLayout, handleError],
  );
};
