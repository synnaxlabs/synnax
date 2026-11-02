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
import { useState } from "react";

export interface AcceptedMethods {
  /** Whether the Core accepts a username and password. */
  password: boolean;
  /** Whether the Core accepts an OpenID Connect provider. */
  oidc: boolean;
}

const DEFAULT_ACCEPTED: AcceptedMethods = { password: true, oidc: false };

interface Read {
  address: string;
  methods: AcceptedMethods;
}

/**
 * Reads the login methods a Core accepts, without a session.
 * @param core - The address of the Core. Undefined reports the default methods.
 * @returns The accepted methods. An unreachable Core, or one older than the methods
 * endpoint, reports passwords only: a connection status reports the first, and the
 * second accepts passwords.
 */
export const useAcceptedMethods = (core?: auth.TokenlessParams): AcceptedMethods => {
  const host = core?.host;
  const port = core?.port;
  const secure = core?.secure;
  const address = `${host}:${port}:${secure}`;
  const [read, setRead] = useState<Read | null>(null);
  useAsyncEffect(
    async (signal) => {
      if (host == null || port == null) return;
      const tokenless = auth.createTokenlessClient({ host, port, secure });
      const methods = await auth.retrieveMethods(tokenless).catch(() => null);
      if (methods == null || signal.aborted) return;
      setRead({
        address,
        methods: {
          password: methods.includes("password"),
          oidc: methods.includes("oidc"),
        },
      });
    },
    [host, port, secure],
  );
  return read?.address === address ? read.methods : DEFAULT_ACCEPTED;
};
