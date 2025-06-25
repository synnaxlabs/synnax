// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { useCallback, useState as useState_ } from "react";

import { useStatusSynchronizer } from "@/hardware/device/synchronizers";
import { useAsyncEffect } from "@/hooks";
import { Synnax } from "@/synnax";

export const useStatus = (key: device.Key): device.Status | undefined => {
  const client = Synnax.use();
  const [status, setStatus] = useState_<device.Status | undefined>(undefined);
  useAsyncEffect(
    async (signal) => {
      if (client == null) return;
      const { status } = await client.hardware.devices.retrieve(key, {
        includeStatus: true,
      });
      if (signal.aborted) return;
      setStatus(status);
    },
    [client],
  );
  const handleStatusChange = useCallback(
    (status: device.Status) => {
      if (status.details.device !== key) return;
      setStatus(status);
    },
    [key],
  );

  useStatusSynchronizer(handleStatusChange);
  return status;
};
