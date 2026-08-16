// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * EngineRef is the mutable engine handle a spec creates via vi.hoisted and flips
 * between tests to drive the web/tauri branches.
 */
export interface EngineRef {
  engine: "web" | "tauri";
}

/**
 * mockRuntimeEngine builds the module shape for mocking the session runtime's ENGINE
 * constant. It spreads the real module and overrides ONLY ENGINE with a getter reading
 * the given ref, so Drift and every other export stay real.
 *
 * vi.mock factories are hoisted above imports, so the spec must reach this helper
 * through a dynamic import inside the factory and create the ref via vi.hoisted:
 *
 *   const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));
 *   vi.mock("@/session/runtime/runtime", async (importOriginal) => {
 *     const { mockRuntimeEngine } = await import("@/testutil/runtime");
 *     return await mockRuntimeEngine(importOriginal, mocks);
 *   });
 */
export const mockRuntimeEngine = async (
  importOriginal: () => Promise<unknown>,
  ref: EngineRef,
): Promise<Record<string, unknown>> => ({
  ...((await importOriginal()) as Record<string, unknown>),
  get ENGINE() {
    return ref.engine;
  },
});
