// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Runtime } from "@/session/runtime";

/**
 * EngineRef is the mutable engine handle a spec creates via vi.hoisted and flips
 * between tests to drive the web/tauri branches.
 */
export interface EngineRef {
  engine: "web" | "tauri";
  /** Overrides CORE_EMBEDDED when set, to drive the Synnax Desktop branches. */
  coreEmbedded?: boolean;
}

/**
 * Mocks the `Runtime` namespace of `@/session/runtime`, so that ENGINE, CORE_EMBEDDED,
 * and isMainWindow read the ref on each call. Every other export stays real. A browser
 * mock reads each export of a module once, so only a namespace property can change
 * between tests. vi.mock factories run before imports: create the ref with vi.hoisted,
 * and load this helper with a dynamic import inside the factory.
 */
export const mockRuntimeEngine = async (
  importOriginal: () => Promise<{ Runtime: typeof Runtime }>,
  ref: EngineRef,
): Promise<{ Runtime: typeof Runtime }> => {
  const { Runtime: original } = await importOriginal();
  return {
    Runtime: {
      ...original,
      get ENGINE() {
        return ref.engine;
      },
      get CORE_EMBEDDED() {
        return ref.coreEmbedded ?? original.CORE_EMBEDDED;
      },
      isMainWindow: () => original.isMainWindow(ref.engine),
    },
  };
};
