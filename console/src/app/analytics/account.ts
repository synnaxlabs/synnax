// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor } from "@synnaxlabs/x";

import { type Analytics } from "@/platform/analytics";
import { Session } from "@/session";

interface WatchParams {
  store: Session.Store;
  sink: Pick<Analytics.Sink, "identify" | "reset">;
}

/**
 * Attributes events to the account the app is signed in to until the returned
 * destructor is called. Signing out, or into another account, resets first, so no
 * event of one account lands on another.
 */
export const watchAccount = ({ store, sink }: WatchParams): destructor.Destructor => {
  let current: string | undefined;
  const follow = (): void => {
    const { user, email } = Session.Account.select(store.getState());
    if (user === current) return;
    if (current != null) sink.reset();
    current = user;
    if (user != null && email != null) sink.identify({ id: user, email });
  };
  const unwatch = store.subscribe(follow);
  follow();
  return unwatch;
};
