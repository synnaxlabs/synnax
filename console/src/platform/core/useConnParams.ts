// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { SynnaxParams } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { deep } from "@synnaxlabs/x";
import { useEffect, useState } from "react";

import { accountOf, readToken } from "@/platform/core/token";
import { Session } from "@/session";

interface StoredToken {
  account: string;
  lastLogin?: number;
  token?: string;
}

/** What a client connects to and logs in as. A change to it needs a new client. */
type Identity = Pick<
  Session.Core.Core,
  "key" | "host" | "port" | "secure" | "oidc" | "lastLogin"
>;

const identityOf = ({
  key,
  host,
  port,
  secure,
  oidc,
  lastLogin,
}: Session.Core.Core): Identity => ({
  key,
  host,
  port,
  secure,
  oidc,
  lastLogin,
});

interface Held {
  identity: Identity;
  params: SynnaxParams;
}

/**
 * Reports whether the Core serves the page itself. Only then does the browser send the
 * session cookie back, so only then may the session live in one.
 */
const servesPage = ({ host, port, secure }: Session.Core.Core): boolean => {
  const origin = new URL(window.location.origin);
  const originPort =
    origin.port !== "" ? Number(origin.port) : origin.protocol === "https:" ? 443 : 80;
  return (
    host === origin.hostname &&
    port === originPort &&
    secure === (origin.protocol === "https:")
  );
};

const paramsOf = (
  core: Session.Core.Core,
  stored: StoredToken | undefined,
): SynnaxParams | undefined => {
  const { host, port, secure, username, password, oidc } = core;
  const params: SynnaxParams =
    oidc == null
      ? {
          host,
          port,
          secure,
          username: username === "" ? undefined : username,
          password: password === "" ? undefined : password,
        }
      : { host, port, secure, oidc };
  if (Session.Runtime.ENGINE !== "tauri") {
    const cookie = servesPage(core);
    return { ...params, sessionCookie: cookie, sessionPersistent: cookie };
  }
  if (stored?.account !== accountOf(core) || stored.lastLogin !== core.lastLogin)
    return undefined;
  return { ...params, token: stored.token, sessionPersistent: true };
};

/**
 * Reports whether a client for the Core has a password, OpenID Connect code, or saved
 * session to log in with. On the web, a Core on another origin keeps its session only
 * in the page's memory, so a reload loses it. On desktop, the keychain may hold one.
 */
export const hasLogin = (core: Session.Core.Core): boolean =>
  Session.Runtime.ENGINE === "tauri" ||
  servesPage(core) ||
  core.password !== "" ||
  core.oidc != null;

/**
 * The parameters a client needs to reach the selected Core. A password or OpenID
 * Connect code the user has just entered logs in; otherwise the saved session resumes,
 * from the keychain on desktop and from the session cookie on a page the Core serves.
 * Undefined while no Core is selected or the selected Core has no login, and, on
 * desktop, until the keychain has answered. A saved session outlives the client. Fields
 * that only describe the Core stay out, so renaming it keeps the connection.
 *
 * The parameters change only for another Core, address, or login. A live client
 * follows a change to its own username or password, so the record's copy of one does
 * not replace the client.
 */
export const useConnParams = (): SynnaxParams | undefined => {
  const core = Session.Core.useSelectSelected();
  const handleError = Status.useErrorHandler();
  const [stored, setStored] = useState<StoredToken>();
  const [held, setHeld] = useState<Held>();
  const account = core == null ? undefined : accountOf(core);
  // a login in any window that saves a session is the cue to read the keychain again
  const lastLogin = core?.lastLogin;
  useEffect(() => {
    if (core == null || Session.Runtime.ENGINE !== "tauri") return;
    let stale = false;
    const read = { account: accountOf(core), lastLogin: core.lastLogin };
    readToken(core).then(
      (token) => {
        if (!stale) setStored({ ...read, token });
      },
      (err: unknown) => {
        if (stale) return;
        setStored(read);
        handleError(err, "Failed to read the saved session");
      },
    );
    return () => {
      stale = true;
    };
  }, [account, lastLogin]);
  // A set during render, behind a guard, makes React retry before it commits, so the
  // provider never sees parameters for an identity they do not belong to.
  if (core == null || !hasLogin(core)) {
    if (held != null) setHeld(undefined);
    return undefined;
  }
  const identity = identityOf(core);
  if (held != null && deep.equal(held.identity, identity)) return held.params;
  const params = paramsOf(core, stored);
  if (params == null) return undefined;
  setHeld({ identity, params });
  return params;
};
