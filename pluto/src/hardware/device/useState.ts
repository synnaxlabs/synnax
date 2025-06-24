// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { useState as useState_ } from "react";

import { useStatusSynchronizer } from "@/hardware/device/synchronizers";
import { useAsyncEffect } from "@/hooks";
import { Synnax } from "@/synnax";

export const useState = (key: device.Key): device.Status | undefined => {
  const client = Synnax.use();
  const [state, setState] = useState_<device.Status | undefined>(undefined);
  useAsyncEffect(
    async (signal) => {
      if (client == null) return;
      const { status } = await client.hardware.devices.retrieve(key, {
        includeStatus: true,
      });
      if (signal.aborted) return;
      setState(status);
    },
    [client],
  );
  useStatusSynchronizer(setState);
  return state;
};
