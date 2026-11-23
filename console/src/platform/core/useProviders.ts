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
 * @returns The providers, or undefined while they load or after their read fails. A
 * failed read reports an error status.
 */
export const useProviders = (
  core: auth.TokenlessParams | undefined,
  enabled: boolean,
): auth.OIDCLogin[] | undefined => {
  const host = core?.host;
  const port = core?.port;
  const secure = core?.secure;
  const key =
    !enabled || host == null || port == null ? null : `${host}:${port}:${secure}`;
  const [found, setFound] = useState<{ key: string; providers: auth.OIDCLogin[] }>();
  const handleError = Status.useErrorHandler();
  useAsyncEffect(
    async (signal) => {
      if (key == null || host == null || port == null) return;
      const tokenless = auth.createTokenlessClient({ host, port, secure });
      handleError(async () => {
        const providers = await auth.retrieveProviders(tokenless);
        if (!signal.aborted) setFound({ key, providers });
      }, "Failed to retrieve the login providers");
    },
    [key],
  );
  if (key == null) return [];
  return found?.key === key ? found.providers : undefined;
};
