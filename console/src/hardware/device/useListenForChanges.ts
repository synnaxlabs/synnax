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
import { strings } from "@synnaxlabs/x";
import { useCallback } from "react";

const PREFIX = "new-device-";

export const useListenForChanges = () => {
  const addStatus = Status.useAdder<device.Device>();
  const handleSet = useCallback(
    (dev: device.Device) => {
      if (dev.configured) return;
      addStatus({
        variant: "info",
        key: `${PREFIX}${dev.key}`,
        message: `New ${dev.model} connected`,
        details: dev,
      });
    },
    [addStatus],
  );
  Device.useSetSynchronizer(handleSet);
};

export const getKeyFromStatus = ({
  key,
}: Status.NotificationSpec): device.Key | null => {
  if (!key.startsWith(PREFIX)) return null;
  return strings.trimPrefix(key, PREFIX);
};
