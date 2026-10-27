// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";
import { useEffect } from "react";

import { accountOf, deleteToken, writeToken } from "@/platform/core/token";
import { Session } from "@/session";

/**
 * Keeps the keychain in step with the connection: saves the session token once the
 * Core accepts it and forgets it once the Core rejects it. Renders nothing, and does
 * nothing on the web, where the session lives in a cookie.
 */
export const TokenKeeper = (): null => {
  const client = Synnax.use();
  const status = Synnax.useConnectionStatus();
  const core = Session.Core.useSelectSelected();
  const handleError = Status.useErrorHandler();
  // for a moment after a Core switch, the status still belongs to the old Core's client
  const current =
    client != null &&
    core != null &&
    accountOf(core) ===
      accountOf({
        host: client.params.host,
        port: Number(client.params.port),
        secure: client.params.secure,
        username: client.params.username ?? "",
      });
  const rejected =
    current && status.variant === "error" && status.details.reason === "auth";
  const token = current && status.details.authenticated ? client.auth.token : undefined;
  useEffect(() => {
    if (core == null || Session.Runtime.ENGINE !== "tauri") return;
    if (rejected)
      handleError(async () => await deleteToken(core), "Failed to forget the session");
    else if (token != null && token !== "")
      handleError(
        async () => await writeToken(core, token),
        "Failed to save the session",
      );
  }, [core, rejected, token, handleError]);
  return null;
};
