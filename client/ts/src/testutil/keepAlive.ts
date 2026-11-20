// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Shortens the idle socket timeout of `fetch` to 1 s, below the Core's 5 s. Undici runs
 * a longer timer up to 1 s late, so a request could reuse a socket as the Core closes
 * it. Await it once in the setup file of a Node suite that reaches a Core.
 */
export const shortenKeepAlive = async (): Promise<void> => {
  // Imported here so that the test utilities load in a browser.
  const { Agent, setGlobalDispatcher } = await import("undici");
  setGlobalDispatcher(new Agent({ keepAliveTimeout: 1000 }));
};
