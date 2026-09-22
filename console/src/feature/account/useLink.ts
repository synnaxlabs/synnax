// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { Status, Synnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { useState } from "react";

import { type Linked, parseLink } from "@/feature/account/handoff";
import { Link } from "@/platform/link";
import { Session } from "@/session";

const FAILED_MESSAGE = "Failed to sign in";

/**
 * Takes the sign-in link the hub opens Synnax Desktop with: applies its token to
 * the embedded Core and stores the link. A link whose state this app did not mint is
 * refused.
 */
export const useLink = (deps: Link.Deps = Link.DEFAULT_DEPS): void => {
  // While early returns are usually bad in hooks, this is fine because the engine is a
  // constant and so the hook will be the exact same for a given runtime.
  if (deps.engine !== "tauri") return;
  const handleError = Status.useErrorHandler();
  const dispatch = Session.useDispatch();
  const store = Session.useStore();
  const client = Synnax.use();
  const [received, setReceived] = useState<Linked | null>(null);

  const receive = (urls: string[]): void => {
    try {
      dispatch(Drift.focusWindow({}));
      if (urls.length === 0) throw new Error("The sign-in link is empty");
      const linked = parseLink(urls[0]);
      const { pending } = Session.Account.select(store.getState());
      if (pending == null || linked.state !== pending)
        throw new Error("This sign-in link was not requested by this app");
      setReceived(linked);
    } catch (e) {
      handleError(e, FAILED_MESSAGE);
    }
  };

  // Handles the case where the app is opened from a link
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

  // Handles the case where the app is open and a link gets called
  useAsyncEffect(async () => await deps.onOpenURL(receive), []);

  // A launch link can land before the embedded Core answers, so the token waits for a
  // client.
  useAsyncEffect(
    async (signal) => {
      if (client == null || received == null) return;
      try {
        await client.license.activate(received.token);
        if (signal.aborted) return;
        const { activation, secret, email } = received;
        dispatch(Session.Account.link({ activation, secret, email }));
      } catch (e) {
        if (signal.aborted) return;
        handleError(e, FAILED_MESSAGE);
      }
      setReceived(null);
    },
    [client, received],
  );
};
