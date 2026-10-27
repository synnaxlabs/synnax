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
import { errors } from "@synnaxlabs/x";
import { useEffect, useRef } from "react";

import { addressOf, deleteToken, writeToken } from "@/platform/core/token";
import { Session } from "@/session";

/**
 * Keeps the keychain in step with the connection: saves the session token once the
 * Core accepts it and forgets it once the Core rejects it. On the web, where the
 * session lives in a cookie, it instead removes a spent OpenID Connect code from the
 * Core's record once the cookie holds the session, so no later client logs in with it.
 * Renders nothing.
 */
export const TokenKeeper = (): null => {
  const client = Synnax.use();
  const status = Synnax.useConnectionStatus();
  const core = Session.Core.useSelectSelected();
  const handleError = Status.useErrorHandler();
  const dispatch = Session.useDispatch();
  // for a moment after a Core switch, the status still belongs to the old Core's client
  const current =
    client != null &&
    core != null &&
    addressOf(core) ===
      addressOf({
        host: client.params.host,
        port: Number(client.params.port),
        secure: client.params.secure,
        username: client.params.username ?? "",
      });
  const rejected =
    current && status.variant === "error" && status.details.reason === "auth";
  const token = current && status.details.authenticated ? client.auth.token : undefined;
  // a later change to the record, such as a login in another window, must not replay an
  // old rejection or save over the token that login just saved
  const handledRejection = useRef<typeof status>(undefined);
  const savedToken = useRef<string>(undefined);
  useEffect(() => {
    if (core == null) return;
    if (Session.Runtime.ENGINE !== "tauri") {
      // a cookie session has an empty token. Only a cookie keeps the session for the
      // client that replaces this one; a Core on another origin keeps it here alone.
      if (token === "" && core.oidc != null)
        dispatch(Session.Core.set({ ...core, oidc: undefined }));
      return;
    }
    if (rejected) {
      if (handledRejection.current === status) return;
      handledRejection.current = status;
      handleError(async () => await deleteToken(core), "Failed to forget the session");
    } else if (token != null && token !== "" && savedToken.current !== token) {
      savedToken.current = token;
      handleError(async () => {
        try {
          await writeToken(core, token);
        } catch (e) {
          // a later update retries a failed write
          if (savedToken.current === token) savedToken.current = undefined;
          throw errors.fromUnknown(e);
        }
      }, "Failed to save the session");
    }
  }, [core, rejected, status, token, handleError]);
  return null;
};
