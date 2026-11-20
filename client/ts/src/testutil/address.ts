// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * @returns the Core that specs connect to: `SYNNAX_TEST_HOST` and `SYNNAX_TEST_PORT`
 * when set, else localhost:9090.
 * @throws where there is no `process`, as in a browser. A spec there passes its own
 * host and port.
 */
export const testCoreAddress = (): { host: string; port: number } => {
  if (typeof process === "undefined")
    throw new Error("no process: pass an explicit host and port");
  return {
    host: process.env.SYNNAX_TEST_HOST ?? "localhost",
    port: Number(process.env.SYNNAX_TEST_PORT ?? 9090),
  };
};
