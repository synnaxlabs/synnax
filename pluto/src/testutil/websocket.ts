// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { WebSocket } from "ws";

// jsdom installs its own `Event` global but reuses Node's native `EventTarget`. Node's
// global `WebSocket` (undici) extends that native `EventTarget`, whose `dispatchEvent`
// rejects any event not constructed from the native `Event` realm. When undici fires
// the "open" event it builds a bare `new Event(...)` resolved against jsdom's replaced
// global `Event`, which the native base then rejects ("the 'event' argument must be an
// instance of Event"). This breaks every test that opens a real WebSocket against a
// live cluster, and cannot be reconciled at the global level because jsdom's DOM nodes
// need that same jsdom `Event`.
//
// The `ws` package implements its own event handling with plain event objects rather
// than the native `EventTarget`/`Event` pair, so it is immune to the realm mismatch.
// Swapping it in for the global `WebSocket` in tests lets freighter's WebSocket client
// talk to a live cluster while leaving jsdom's DOM untouched.
//
// installTestWebSocket runs unconditionally for every spec file, so no test ever
// depends on the native/undici WebSocket being present. The swap is therefore a plain,
// permanent global assignment rather than a `vi.stubGlobal` — a stub gets reverted to
// the crashing native implementation at every file boundary, and live-cluster clients
// closing at that same boundary (their own `afterAll`) can have a connection still
// completing its handshake right as the revert happens, crashing the whole run.
export const installTestWebSocket = (): void => {
  globalThis.WebSocket = WebSocket as unknown as typeof globalThis.WebSocket;
};
