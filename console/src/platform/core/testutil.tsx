// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { RenderResult } from "@testing-library/react";
import type { ReactNode } from "react";
import { type MockInstance, onTestFinished, vi } from "vitest";

import { Modals } from "@/platform/modals";
import {
  type ConsolePreloadedState,
  renderWithConsole,
  type TestStore,
} from "@/testutil";

/**
 * Renders ui together with a live {@link Modals.Stack} inside the full console provider
 * stack, so modals opened during the test actually mount. Returns the backing store so
 * a spec can inspect dispatched Core state.
 */
export const renderCoreUI = async (
  ui: ReactNode,
  preloadedState?: ConsolePreloadedState,
): Promise<RenderResult & { store: TestStore }> =>
  await renderWithConsole(
    <>
      {ui}
      <Modals.Stack />
    </>,
    { preloadedState },
  );

/**
 * Answers every request for a Core's login methods with the given methods, because no
 * test Core turns passwords off. Other requests reach their Core.
 * @returns The fetch spy, which restores itself when the test finishes.
 */
export const answerMethods = (methods: string[]): MockInstance<typeof fetch> => {
  const real = globalThis.fetch;
  const spy = vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
    const url = input instanceof Request ? input.url : input.toString();
    if (!url.endsWith("/auth/methods")) return await real(input, init);
    const body = JSON.stringify({ methods, providers: [] });
    return new Response(body, { headers: { "Content-Type": "application/json" } });
  });
  onTestFinished(() => spy.mockRestore());
  return spy;
};
