// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax } from "@synnaxlabs/pluto";
import { useEffect, useEffectEvent } from "react";

/**
 * Runs sweep whenever the change stream connects or reconnects, and once on
 * mount when it is already live. Epoch events only fire on a healthy stream,
 * so sweeps never run while disconnected.
 */
export const useEpochSweep = (sweep: () => Promise<void>): void => {
  const client = Synnax.use();
  const run = useEffectEvent(() => {
    sweep().catch((err: unknown) => console.error("epoch sweep failed", err));
  });
  useEffect(() => {
    if (client == null) return;
    if (client.cache.epoch > 0) run();
    return client.cache.onEpoch(() => run());
  }, [client]);
};
