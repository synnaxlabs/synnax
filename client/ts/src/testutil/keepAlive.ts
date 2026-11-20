// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Agent, setGlobalDispatcher } from "undici";

/**
 * Shortens how long `fetch` keeps an idle socket to 1 s. The Core closes an idle socket
 * after 5 s. Undici keeps one for 4 s on a timer that can run about 1 s late, so a
 * request can reuse a socket as it closes. Only timeouts of 1 s or less use a precise
 * timer. Call it once from the setup file of a Node suite that reaches a Core.
 */
export const shortenKeepAlive = (): void =>
  setGlobalDispatcher(new Agent({ keepAliveTimeout: 1000 }));
