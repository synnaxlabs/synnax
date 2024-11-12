// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Status, Synnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { type change } from "@synnaxlabs/x";

export const useListenForChanges = (): void => {
  const client = Synnax.use();
  const addStatus = Status.useAggregator();
  useAsyncEffect(async () => {
    if (client == null) return;
    const tracker = await client.hardware.devices.openDeviceTracker();
    tracker.onChange((changes) => {
      const sets = changes.filter(({ variant }) => variant === "set") as Array<
        change.Set<string, device.Device>
      >;
      sets.forEach(({ value: dev }) => {
        if (dev.configured === true) return;
        addStatus({
          variant: "info",
          message: `New ${dev.model} connected`,
          data: dev,
        });
      });
    });
    return () => void tracker.close();
  }, [client, addStatus]);
};
