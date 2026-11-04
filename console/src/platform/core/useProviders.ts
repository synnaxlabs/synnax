// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { auth } from "@synnaxlabs/client";
import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Status } from "@synnaxlabs/lyra/status";
import { useState } from "react";

/**
 * Reads where to log in through each provider of a Core, without a session.
 * @param core - The address of the Core.
 * @param enabled - Whether to read them. A false value reports no providers.
 * @returns The providers. A failed read reports an error status and no providers.
 */
export const useProviders = (
  core: auth.TokenlessParams | undefined,
  enabled: boolean,
): auth.OIDCLogin[] => {
  const host = core?.host;
  const port = core?.port;
  const secure = core?.secure;
  const [providers, setProviders] = useState<auth.OIDCLogin[]>([]);
  const handleError = Status.useErrorHandler();
  useAsyncEffect(
    async (signal) => {
      setProviders([]);
      if (!enabled || host == null || port == null) return;
      const tokenless = auth.createTokenlessClient({ host, port, secure });
      handleError(async () => {
        const retrieved = await auth.retrieveProviders(tokenless);
        if (!signal.aborted) setProviders(retrieved);
      }, "Failed to retrieve the login providers");
    },
    [host, port, secure, enabled],
  );
  return providers;
};
