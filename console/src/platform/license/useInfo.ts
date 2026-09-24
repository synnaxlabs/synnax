// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type license } from "@synnaxlabs/client";
import { Synnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { useState } from "react";

export interface InfoResult {
  /** The Core's license state, once retrieved. */
  info?: license.Info;
  /** Why the state could not be retrieved. */
  error?: Error;
}

/**
 * Retrieves the active Core's license state. Re-reads when the client changes or the
 * connection reaches a new epoch, so an activation elsewhere is picked up.
 */
export const useInfo = (): InfoResult => {
  const client = Synnax.use();
  const { details } = Synnax.useConnectionStatus();
  const [result, setResult] = useState<InfoResult>({});
  useAsyncEffect(
    async (signal) => {
      if (client == null) return setResult({});
      try {
        const info = await client.license.retrieve();
        if (!signal.aborted) setResult({ info });
      } catch (error) {
        if (signal.aborted) return;
        setResult({ error: error instanceof Error ? error : new Error(String(error)) });
      }
    },
    [client, details.epoch],
  );
  return result;
};
