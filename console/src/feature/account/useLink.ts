// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type license } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";
import { useState } from "react";

import { type Linked, parseLink } from "@/feature/account/handoff";
import { Link } from "@/platform/link";
import { Session } from "@/session";

const FAILED_MESSAGE = "Failed to log in";

export interface LinkDeps extends Link.Deps {
  /** The Core's license API. Defaults to the connected client's. */
  license?: Pick<license.Client, "activate">;
}

/**
 * Takes the login link the hub opens Synnax Desktop with: applies its license key to
 * the embedded Core and stores the link. A link whose state this app did not mint is
 * refused.
 */
export const useLink = (deps: LinkDeps = Link.DEFAULT_DEPS): void => {
  // The engine is fixed per runtime, so this early return never changes the hook order.
  if (deps.engine !== "tauri") return;
  const handleError = Status.useErrorHandler();
  const dispatch = Session.useDispatch();
  const store = Session.useStore();
  const client = Synnax.use();
  const api = deps.license ?? client?.license;
  const [received, setReceived] = useState<Linked | null>(null);

  const receive = (urls: string[]): void => {
    try {
      dispatch(Drift.focusWindow({}));
      if (urls.length === 0) throw new Error("The login link is empty");
      const linked = parseLink(urls[0]);
      const pending = Session.Account.selectPending(store.getState());
      if (pending == null || linked.state !== pending)
        throw new Error("This login link was not requested by this app");
      setReceived(linked);
    } catch (e) {
      handleError(e, FAILED_MESSAGE);
    }
  };

  useAsyncEffect(async (signal) => {
    const urls = await deps.getCurrentURLs();
    // A hard reload re-runs this effect with the same launch link; skip it once.
    if (localStorage.getItem(Link.SHOULD_IGNORE_KEY) === "true") {
      localStorage.setItem(Link.SHOULD_IGNORE_KEY, "false");
      return;
    }
    if (urls == null || signal.aborted) return;
    receive(urls);
  }, []);

  useAsyncEffect(async () => await deps.onOpenURL(receive), []);

  // A launch link can land before the embedded Core answers, so the license key waits
  // for a client.
  useAsyncEffect(
    async (signal) => {
      if (api == null || received == null) return;
      try {
        await api.activate(received.key);
        if (signal.aborted) return;
        const { secret, email } = received;
        dispatch(Session.Account.link({ secret, email }));
      } catch (e) {
        if (signal.aborted) return;
        handleError(e, FAILED_MESSAGE);
      }
      setReceived(null);
    },
    [api, received],
  );
};
