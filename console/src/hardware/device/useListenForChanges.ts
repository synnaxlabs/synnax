// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Status, Synnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { type UnknownRecord } from "@synnaxlabs/x";

const PREFIX = "new-device-";

export const useListenForChanges = () => {
  const client = Synnax.use();
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
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
            key: `${PREFIX}${device.key}`,
            message: `New ${device.model} connected`,
            data: device as unknown as UnknownRecord,
          });
        });
    });
    return () => {
      tracker.close().catch((e) => handleError(e, "Failed to close device tracker"));
    };
  }, [addStatus, client, handleError]);
};

const PREFIX_LENGTH = PREFIX.length;

export const getKeyFromStatus = ({
  key,
}: Status.NotificationSpec): device.Key | null =>
  key.startsWith(PREFIX) ? key.slice(PREFIX_LENGTH) : null;
