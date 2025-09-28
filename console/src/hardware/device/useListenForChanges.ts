// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Device, Status } from "@synnaxlabs/pluto";
import { useCallback, useRef } from "react";

export const useListenForChanges = () => {
  const addStatus = Status.useAdder();
  const alreadyNotified = useRef<Set<device.Key>>(new Set());
  const handleSet = useCallback(
    (dev: device.Device) => {
      if (dev.configured || alreadyNotified.current.has(dev.key)) return;
      alreadyNotified.current.add(dev.key);
      addStatus<device.Device>({
        variant: "info",
        message: `New ${dev.model} connected`,
        details: dev,
      });
    },
    [addStatus],
  );
  Device.useSetSynchronizer(handleSet);
};

export const getKeyFromStatus = ({
  details,
}: Status.NotificationSpec<typeof device.deviceZ>): device.Key | null => {
  if (details == null || details.configured || !("key" in details)) return null;
  return details.key;
};
