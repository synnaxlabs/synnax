// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { inject } from "vitest";

declare module "vitest" {
  export interface ProvidedContext {
    /** The Core that browser specs connect to, as Node read it. */
    synnaxTestCore?: { host: string; port: number };
  }
}

/**
 * @returns the Core that specs connect to: `SYNNAX_TEST_HOST` and `SYNNAX_TEST_PORT`
 * when set, else localhost:9090. In a browser, it is the `synnaxTestCore` that the
 * Vitest project provides.
 * @throws in a browser whose Vitest project provides no `synnaxTestCore`.
 */
export const testCoreAddress = (): { host: string; port: number } => {
  if (typeof process !== "undefined")
    return {
      host: process.env.SYNNAX_TEST_HOST ?? "localhost",
      port: Number(process.env.SYNNAX_TEST_PORT ?? 9090),
    };
  const core = inject("synnaxTestCore");
  if (core == null)
    throw new Error(
      "no Core address in this browser: provide synnaxTestCore in the Vitest project",
    );
  return core;
};
