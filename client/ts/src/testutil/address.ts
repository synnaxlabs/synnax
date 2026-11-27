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
    /** The Core that live-Core specs connect to. */
    synnaxTestCore?: { host: string; port: number };
  }
}

/**
 * @returns the Core that specs connect to, as the Vitest project provides it in
 * `synnaxTestCore`.
 * @throws in a Vitest project that provides no `synnaxTestCore`.
 */
export const testCoreAddress = (): { host: string; port: number } => {
  const core = inject("synnaxTestCore");
  if (core == null)
    throw new Error(
      "no test Core address: provide synnaxTestCore in the Vitest project",
    );
  return core;
};
