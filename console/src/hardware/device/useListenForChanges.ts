// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status, Synnax, useAsyncEffect } from "@synnaxlabs/pluto";

export const NEW_STATUS_KEY_PREFIX = "new-device-";

export const useListenForChanges = (): void => {
  const client = Synnax.use();
  const addStatus = Status.useAggregator();
  useAsyncEffect(async () => {
    if (client == null) return;
    const tracker = await client.hardware.devices.openDeviceTracker();
    tracker.onChange((changes) => {
      changes
        .filter((c) => c.variant === "set")
        .forEach(({ value: device }) => {
          if (device.configured) return;
          addStatus({
            variant: "info",
            key: `${NEW_STATUS_KEY_PREFIX}${device.key}`,
            message: `New ${device.model} connected`,
            data: device,
          });
        });
    });
    return () => void tracker.close();
  }, [addStatus, client]);
};
