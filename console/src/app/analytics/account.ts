// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor } from "@synnaxlabs/x";

import { type Transport } from "@/app/analytics/posthog";
import { type Analytics } from "@/platform/analytics";
import { Session } from "@/session";

interface WatchParams {
  store: Session.Store;
  sink: Pick<Analytics.Sink, "identify" | "reset">;
  transport: Pick<Transport, "start" | "stop">;
}

/**
 * Runs the transport only while the user allows analytics, and attributes events to the
 * account the app is signed in to until the returned destructor is called. Signing out,
 * or into another account, resets first, so no event of one account lands on another.
 */
export const watchAccount = ({
  store,
  sink,
  transport,
}: WatchParams): destructor.Destructor => {
  let allowed = false;
  let current: string | undefined;
  const follow = (): void => {
    const { link, analyticsAllowed } = Session.Account.selectSliceState(
      store.getState(),
    );
    if (analyticsAllowed !== allowed) {
      allowed = analyticsAllowed;
      // A stop resets the vendor, so the next start identifies the account again.
      current = undefined;
      if (allowed) transport.start();
      else transport.stop();
    }
    if (!allowed || link?.user === current) return;
    if (current != null) sink.reset();
    current = link?.user;
    if (link != null) sink.identify({ id: link.user, email: link.email });
  };
  const unwatch = store.subscribe(follow);
  follow();
  return unwatch;
};
