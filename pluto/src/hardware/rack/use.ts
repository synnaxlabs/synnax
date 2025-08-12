// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack } from "@synnaxlabs/client";
import { useCallback, useState } from "react";

import { useStatusSynchronizer } from "@/hardware/rack/synchronizers";
import { useAsyncEffect } from "@/hooks";
import { Synnax } from "@/synnax";

export const use = (key: rack.Key): rack.Rack | undefined => {
  const client = Synnax.use();
  const [rack, setRack] = useState<rack.Rack | undefined>(undefined);
  useAsyncEffect(
    async (signal) => {
      if (client == null || key === 0) {
        setRack(undefined);
        return;
      }
      const rack = await client.hardware.racks.retrieve({ key });
      if (signal.aborted) return;
      setRack(rack);
    },
    [client, key],
  );
  const handleStatusChange = useCallback(
    (status: rack.Status) => {
      if (rack == null || client == null) return;
      setRack(client.hardware.racks.sugar({ ...rack.payload, status }));
    },
    [rack, client],
  );
  useStatusSynchronizer(handleStatusChange);
  return rack;
};
