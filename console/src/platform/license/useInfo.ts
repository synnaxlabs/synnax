// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type license } from "@synnaxlabs/client";
import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Synnax } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useState } from "react";

export interface InfoResult {
  info?: license.Info;
  error?: Error;
}

/**
 * Retrieves the active Core's license state. Re-reads when the client changes or the
 * connection reaches a new epoch.
 */
export const useInfo = (): InfoResult => {
  const client = Synnax.use();
  const { details } = Synnax.useConnectionStatus();
  const [result, setResult] = useState<InfoResult>({});
  useAsyncEffect(
    async (signal) => {
      if (client == null) {
        setResult({});
        return;
      }
      try {
        const info = await client.license.retrieve();
        if (!signal.aborted) setResult({ info });
      } catch (error) {
        if (signal.aborted) return;
        setResult({ error: errors.fromUnknown(error) });
      }
    },
    [client, details.epoch],
  );
  return result;
};
