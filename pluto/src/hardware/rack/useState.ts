// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack } from "@synnaxlabs/client";
import { useState } from "react";

import { useStatusSynchronizer } from "@/hardware/rack/synchronizers";
import { useAsyncEffect } from "@/hooks";
import { Synnax } from "@/synnax";

export const useStatus = (key: rack.Key): rack.Status | undefined => {
  const client = Synnax.use();
  const [status, setStatus] = useState<rack.Status | undefined>(undefined);
  useAsyncEffect(
    async (signal) => {
      if (client == null) return;
      const { status } = await client.hardware.racks.retrieve(key, {
        includeStatus: true,
      });
      if (signal.aborted) return;
      setStatus(status);
    },
    [client],
  );
  useStatusSynchronizer(setStatus);
  return status;
};
