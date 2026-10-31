// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type SynnaxParams } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { useEffect, useState } from "react";

import { accountOf, readToken } from "@/platform/core/token";
import { Session } from "@/session";

interface StoredToken {
  account: string;
  token?: string;
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

/**
 * The parameters a client needs to reach the selected Core. A password the user has
 * just entered logs in; otherwise the saved session resumes, from the keychain on
 * desktop and from the session cookie on a page the Core serves. Undefined while no
 * Core is selected and, on desktop, until the keychain has answered. A saved session
 * outlives the client. Fields that only describe the Core stay out, so renaming it
 * keeps the connection.
 */
export const useConnParams = (): SynnaxParams | undefined => {
  const core = Session.Core.useSelectSelected();
  const handleError = Status.useErrorHandler();
  const [stored, setStored] = useState<StoredToken>();
  const account = core == null ? undefined : accountOf(core);
  useEffect(() => {
    if (core == null || Session.Runtime.ENGINE !== "tauri") return;
    let stale = false;
    readToken(core).then(
      (token) => {
        if (!stale) setStored({ account: accountOf(core), token });
      },
      (err: unknown) => {
        if (stale) return;
        setStored({ account: accountOf(core) });
        handleError(err, "Failed to read the saved session");
      },
    );
    return () => {
      stale = true;
    };
  }, [account]);
  if (core == null) return undefined;
  const { host, port, secure, username, password } = core;
  const params: SynnaxParams = {
    host,
    port,
    secure,
    username,
    password: password === "" ? undefined : password,
  };
  if (Session.Runtime.ENGINE !== "tauri") {
    const cookie = servesPage(core);
    return { ...params, sessionCookie: cookie, sessionPersistent: cookie };
  }
  if (stored == null || stored.account !== account) return undefined;
  return { ...params, token: stored.token, sessionPersistent: true };
};
