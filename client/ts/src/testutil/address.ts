// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// A browser has no process, so specs there connect to the defaults.
const env: Partial<Record<string, string>> =
  typeof process === "undefined" ? {} : process.env;

/**
 * @returns the Core that specs connect to: `SYNNAX_TEST_HOST` and `SYNNAX_TEST_PORT`
 * when set, else localhost:9090.
 */
export const testCoreAddress = (): { host: string; port: number } => ({
  host: env.SYNNAX_TEST_HOST ?? "localhost",
  port: Number(env.SYNNAX_TEST_PORT ?? 9090),
});
