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
import { useEffect, useMemo, useState } from "react";

import { accountOf, readToken } from "@/platform/core/token";
import { Session } from "@/session";

interface StoredToken {
  account: string;
  token?: string;
}

/**
 * The parameters a client needs to reach the selected Core. A password the user has
 * just entered logs in; otherwise the saved session resumes, from the keychain on
 * desktop and from the session cookie on the web. Undefined while no Core is selected
 * and, on desktop, until the keychain has answered.
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
  return useMemo(() => {
    if (core == null) return undefined;
    const params = {
      ...core,
      password: core.password === "" ? undefined : core.password,
    };
    if (Session.Runtime.ENGINE !== "tauri") return { ...params, sessionCookie: true };
    if (stored == null || stored.account !== account) return undefined;
    return { ...params, token: stored.token };
  }, [core, stored, account]);
};
