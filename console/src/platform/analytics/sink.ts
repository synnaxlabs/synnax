// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Account,
  accountZ,
  type Name,
  type Properties,
  schemas,
} from "@/platform/analytics/events";

/**
 * The host of a synthesized screen address. The Console has no addresses, so screens
 * borrow one: a host of its own separates the app from the website inside the shared
 * project, and reads as a page name in the dashboards.
 */
const HOST = "desktop.synnaxlabs.com";

/**
 * A tab type is an identifier someone wrote in the source. A name someone typed into
 * the app almost never survives this, so a panel name reaching {@link Sink.screen} by
 * mistake is dropped instead of reported.
 */
const TYPE = /^[A-Za-z0-9_]+$/;

/** What a vendor supplies. Every argument has already been checked. */
export interface Transport {
  capture: (event: string, properties: Record<string, unknown>) => void;
  identify: (account: Account) => void;
  reset: () => void;
}

/** What the rest of the Console calls. No method throws. */
export interface Sink {
  capture: <N extends Name>(event: N, properties: Properties<N>) => void;
  /** Records a view of the given tab type as a screen. */
  screen: (tab: string) => void;
  /** Attributes this and every later event to the account. */
  identify: (account: Account) => void;
  /** Forgets the account, so later events belong to no one until the next identify. */
  reset: () => void;
}

/** Discards everything. The Console build keeps it. */
export const NOOP: Sink = {
  capture: () => {},
  screen: () => {},
  identify: () => {},
  reset: () => {},
};

/**
 * Runs a call into the vendor. Analytics must never fail the action that reported it,
 * so a vendor that throws is logged instead.
 */
const send = (call: () => void): void => {
  try {
    call();
  } catch (err) {
    console.error("failed to send analytics", err);
  }
};

/**
 * Wraps a transport in the event contract. Anything the schemas do not declare is
 * dropped and logged instead of sent, so an unreviewed property cannot reach a vendor
 * even if a caller type-asserts past the signature.
 */
export const createSink = (transport: Transport): Sink => ({
  capture: (event, properties) => {
    const parsed = schemas[event].safeParse(properties);
    if (!parsed.success) {
      console.error(`dropped ${event}`, parsed.error);
      return;
    }
    send(() => transport.capture(event, parsed.data));
  },
  screen: (tab) => {
    if (!TYPE.test(tab)) {
      console.error(`dropped a screen for ${tab}`);
      return;
    }
    // All three are overridden together. Left alone they describe the `tauri://`
    // origin, which every screen shares.
    send(() =>
      transport.capture("$pageview", {
        $current_url: `https://${HOST}/${tab}`,
        $host: HOST,
        $pathname: `/${tab}`,
      }),
    );
  },
  identify: (account) => {
    const parsed = accountZ.safeParse(account);
    if (!parsed.success) {
      console.error("dropped the account", parsed.error);
      return;
    }
    send(() => transport.identify(parsed.data));
  },
  reset: () => send(() => transport.reset()),
});
