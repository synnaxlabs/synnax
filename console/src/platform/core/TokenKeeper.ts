// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Synnax as Client } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useEffect, useRef, useState } from "react";

import { deleteToken, readToken, writeToken } from "@/platform/core/token";
import { Session } from "@/session";

/**
 * Keeps the saved session token in step with the connection: saves it once the Core
 * accepts it and forgets it once the Core rejects it. When a newer token is saved, such
 * as one another window saved after its user set their own password, the client
 * resumes that session instead. On a page a Core serves, where the session lives in a
 * cookie, it instead removes a spent OpenID Connect code from the Core's record once
 * the cookie holds the session, so no later client logs in with it. Renders nothing.
 */
export const TokenKeeper = (): null => {
  const client = Synnax.use();
  const status = Synnax.useConnectionStatus();
  const core = Session.Core.useSelectSelected();
  const handleError = Status.useErrorHandler();
  const dispatch = Session.useDispatch();
  // The saved Core selected when the client appeared owns the client. Two saved Cores
  // can share an address, so the address alone cannot tell them apart.
  const [owner, setOwner] = useState<{ client: Client; key: string }>();
  if (client != null && core != null && owner?.client !== client)
    setOwner({ client, key: core.key });
  // for a moment after a Core switch, the status still belongs to the old Core's client
  const current =
    client != null &&
    core != null &&
    owner?.client === client &&
    owner.key === core.key &&
    // the client outlives a rename of its user, so the username stays out
    core.host === client.params.host &&
    core.port === Number(client.params.port) &&
    core.secure === client.params.secure;
  const rejected =
    current && status.variant === "error" && status.details.reason === "auth";
  const token = current && status.details.authenticated ? client.auth.token : undefined;
  // a later change to the record, such as a login in another window, must not replay an
  // old rejection or save over the token that login just saved
  const handledRejection = useRef<typeof status>(undefined);
  const savedToken = useRef<string>(undefined);
  // the last token this client held, to tell a rejected token from a newer saved one
  const held = useRef<{ client: Client; token: string }>(undefined);
  useEffect(() => {
    if (core == null) return;
    if (client != null && token != null && token !== "")
      held.current = { client, token };
    if (!Session.Runtime.CORE_SELECTABLE) {
      // a cookie session has an empty token. Only a cookie keeps the session for the
      // client that replaces this one; a Core on another origin keeps it here alone.
      if (token === "" && core.oidc != null)
        dispatch(Session.Core.set({ ...core, oidc: undefined }));
      return;
    }
    if (rejected && client != null) {
      if (handledRejection.current === status) return;
      handledRejection.current = status;
      const last = held.current?.client === client ? held.current.token : undefined;
      handleError(async () => {
        const saved = await readToken(core);
        if (saved == null) return;
        if (saved === last) {
          await deleteToken(core, saved);
          return;
        }
        held.current = { client, token: saved };
        client.resume(saved);
      }, "Failed to forget the session");
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
        // a spent code outranks a saved token, so the next client would log in with it
        if (core.oidc != null)
          dispatch(Session.Core.recordLogin({ key: core.key, at: Date.now() }));
      }, "Failed to save the session");
    }
  }, [client, core, rejected, status, token, handleError]);
  return null;
};
