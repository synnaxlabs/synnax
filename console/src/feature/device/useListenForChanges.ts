// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device } from "@synnaxlabs/client";
import { Device, Status } from "@synnaxlabs/pluto";
import { useCallback, useRef } from "react";

export const useListenForChanges = () => {
  const addStatus = Status.useAdder();
  const alreadyNotified = useRef<Set<device.Key>>(new Set());
  const handleSet = useCallback(
    (dev: device.Device) => {
      if (dev.configured || alreadyNotified.current.has(dev.key)) return;
      alreadyNotified.current.add(dev.key);
      addStatus<ReturnType<typeof device.deviceZ>>({
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
}: {
  details?: unknown;
}): device.Key | null => {
  const parsed = device.deviceZ().safeParse(details);
  if (!parsed.success || parsed.data.configured) return null;
  return parsed.data.key;
};
