// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack } from "@synnaxlabs/client";
import { useState as useState_ } from "react";

import { useStateSynchronizer } from "@/hardware/rack/synchronizers";
import { useAsyncEffect } from "@/hooks/useAsyncEffect";
import { Synnax } from "@/synnax";

export const useState = (key: rack.Key): rack.State | undefined => {
  const client = Synnax.use();
  const [state, setState] = useState_<rack.State | undefined>(undefined);
  useAsyncEffect(
    async (signal) => {
      if (client == null) return;
      const { state } = await client.hardware.racks.retrieve(key, {
        includeState: true,
      });
      if (signal.aborted) return;
      setState(state);
    },
    [client],
  );
  useStateSynchronizer(setState);
  return state;
};
