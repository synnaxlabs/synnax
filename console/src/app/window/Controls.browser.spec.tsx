// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { resolve } from "@synnaxlabs/lyra/styletest";
import { clearMocks, mockIPC, mockWindows } from "@tauri-apps/api/mocks";
import { screen } from "@testing-library/react";
import { describe, expect, it, onTestFinished, vi } from "vitest";
import { page, server, userEvent } from "vitest/browser";

// The Console draws its own window controls on Windows alone.
await vi.hoisted(async () => {
  const { pinOS } = await import("@/testutil/pinOS");
  pinOS("Windows");
});

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "tauri" }));

vi.mock("@/session/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Window } from "@/app/window";
import { Session } from "@/session";
import {
  type ConsolePreloadedState,
  installPortalRoot,
  renderWithConsole,
} from "@/testutil";

const [CORE_KEY] = Object.keys(Session.Core.ZERO_SLICE_STATE.cores);

const WORKSPACE: ConsolePreloadedState = {
  [Session.Core.SLICE_NAME]: { ...Session.Core.ZERO_SLICE_STATE, selected: CORE_KEY },
  [Session.Project.SLICE_NAME]: {
    ...Session.Project.ZERO_SLICE_STATE,
    selected: "f5d0a5c8-5a1e-4b1f-9c0e-0b0c1d2e3f40",
  },
};

const renderPrimary = async (
  preloadedState?: ConsolePreloadedState,
): Promise<HTMLElement> => {
  // The default viewport is narrower than the Console's top bar.
  const { width, height } = server.config.browser.viewport;
  onTestFinished(async () => await page.viewport(width, height));
  await page.viewport(1280, 800);
  // A dev build version skips the update check.
  mockWindows(Drift.MAIN_WINDOW);
  mockIPC((cmd) => (cmd === "plugin:app|version" ? "0.0.0" : undefined));
  onTestFinished(clearMocks);
  await renderWithConsole(<Window.Window />, {
    preloadedState,
    // No Core is connected, so the panel boundary catches the disconnect.
    onCaughtError: () => {},
  });
  return await screen.findByRole("button", { name: "Close" });
};

const buttonAt = (x: number, y: number): string | null =>
  document.elementFromPoint(x, y)?.closest("button")?.getAttribute("aria-label") ??
  null;

// A throw to the window's corner closes it, and a throw to its top edge hits the button
// below.
const expectEdgesReachButtons = (): void => {
  const right = document.documentElement.clientWidth - 1;
  expect(buttonAt(right, 0)).toBe("Close");
  const minimize = screen.getByRole("button", { name: "Minimize" });
  const { left, width } = minimize.getBoundingClientRect();
  expect(buttonAt(left + width / 2, 0)).toBe("Minimize");
};

describe("Window controls on Windows", () => {
  installPortalRoot();

  it("should reach the window's edges from the top bar", async () => {
    await renderPrimary(WORKSPACE);
    expectEdgesReachButtons();
  });

  it("should reach the window's edges from an island before a workspace", async () => {
    await renderPrimary();
    expectEdgesReachButtons();
  });

  it("should turn close red with a white glyph on hover", async () => {
    const close = await renderPrimary(WORKSPACE);
    await userEvent.hover(close);
    expect(getComputedStyle(close).color).toBe(resolve("var(--pluto-white)"));
    expect(getComputedStyle(close, "::before").backgroundColor).toBe(
      resolve("var(--pluto-error-m1)", "background-color"),
    );
    // Only the pill fills, so no square frame shows around it.
    expect(getComputedStyle(close).backgroundColor).toBe(
      resolve("transparent", "background-color"),
    );
    expect((await screen.findByRole("tooltip")).textContent).toBe("Close");
  });
});
