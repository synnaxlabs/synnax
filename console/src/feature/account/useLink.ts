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

import { parseLink } from "@/feature/account/handoff";
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
 * refused. The session store holds the link until its license applies, so a reload or a
 * restart does not lose the login.
 */
export const useLink = (deps: LinkDeps = Link.DEFAULT_DEPS): void => {
  const handleError = Status.useErrorHandler();
  const dispatch = Session.useDispatch();
  const store = Session.useStore();
  const client = Synnax.use();
  const api = deps.license ?? client?.license;
  const received = Session.Account.useSelectReceived();

  const receive = (urls: string[]): void =>
    handleError(() => {
      dispatch(Drift.focusWindow({}));
      if (urls.length === 0) throw new Error("The login link is empty");
      const { state, key, secret, email, user, analyticsAllowed } = parseLink(urls[0]);
      const pending = Session.Account.selectPending(store.getState());
      if (pending == null || state !== pending)
        throw new Error("This login link was not requested by this app");
      dispatch(Session.Account.receive({ key, secret, email, user, analyticsAllowed }));
    }, FAILED_MESSAGE);

  Link.useReceive(receive, deps);

  // A launch link can land before the embedded Core answers, so the license key waits
  // for a client.
  useAsyncEffect(
    async (signal) => {
      if (api == null || received == null) return;
      const { key, ...next } = received;
      try {
        await api.activate(key);
        if (signal.aborted) return;
        dispatch(Session.Account.link(next));
      } catch (e) {
        if (signal.aborted) return;
        handleError(e, FAILED_MESSAGE);
        dispatch(Session.Account.discardReceived());
      }
    },
    [api, received],
  );
};
