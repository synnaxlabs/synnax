// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { auth } from "@synnaxlabs/client";
import { fireEvent, type RenderResult, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { type MockInstance, onTestFinished, vi } from "vitest";

import { Modals } from "@/platform/modals";
import {
  type ConsolePreloadedState,
  getBySelector,
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
 * Answers every request for a Core's login methods with the given methods and
 * providers, because no test Core turns passwords off or accepts a provider. Other
 * requests reach their Core.
 * @returns The fetch spy, which restores itself when the test finishes.
 */
export const answerMethods = (
  methods: string[],
  providers: auth.OIDCLogin[] = [],
): MockInstance<typeof fetch> => {
  const real = globalThis.fetch;
  const spy = vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
    const url = input instanceof Request ? input.url : input.toString();
    if (!url.endsWith("/auth/methods")) return await real(input, init);
    const body = JSON.stringify({ methods, providers });
    return new Response(body, { headers: { "Content-Type": "application/json" } });
  });
  onTestFinished(() => spy.mockRestore());
  return spy;
};

/** The name field of the open connect modal. */
export const connectNameInput = (): HTMLInputElement =>
  screen.getByPlaceholderText<HTMLInputElement>("Synnax Core");

// The port field carries its default rather than a placeholder, so it is reached by
// the field class instead.
const connectPortInput = (): HTMLInputElement =>
  getBySelector<HTMLInputElement>(document.body, ".lyra-field__port input");

/** Fills the open connect modal's name, host, and port fields. */
export const fillConnectForm = (name: string, host: string, port: string): void => {
  fireEvent.change(connectNameInput(), { target: { value: name } });
  fireEvent.change(screen.getByPlaceholderText("localhost"), {
    target: { value: host },
  });
  fireEvent.change(connectPortInput(), { target: { value: port } });
  // A numeric input commits on blur, not per keystroke.
  fireEvent.blur(connectPortInput());
};
