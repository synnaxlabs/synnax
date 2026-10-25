// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { act } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Window } from "@/platform/window";
import { renderWithConsole } from "@/testutil";

const NATIVE_CONTROLS = ".console-window__native-controls";

describe("Window.Controls", () => {
  beforeEach(() => {
    mocks.engine = "web";
  });

  it("renders nothing outside the tauri engine", async () => {
    const { container } = await renderWithConsole(<Window.Controls forceOS="macOS" />);
    expect(container.querySelector(NATIVE_CONTROLS)).toBeNull();
    expect(container.querySelector("button")).toBeNull();
  });

  it("holds the place of the native controls on macOS", async () => {
    mocks.engine = "tauri";
    const { container } = await renderWithConsole(<Window.Controls forceOS="macOS" />);
    expect(container.querySelector(NATIVE_CONTROLS)).not.toBeNull();
    expect(container.querySelector("button")).toBeNull();
  });

  it("renders the window controls on Windows", async () => {
    mocks.engine = "tauri";
    const { container } = await renderWithConsole(
      <Window.Controls forceOS="Windows" />,
    );
    expect(container.querySelectorAll("button")).toHaveLength(3);
    expect(container.querySelector(NATIVE_CONTROLS)).toBeNull();
  });

  it("renders nothing when visibleIfOS does not match the os", async () => {
    mocks.engine = "tauri";
    const { container } = await renderWithConsole(
      <Window.Controls forceOS="macOS" visibleIfOS="Windows" />,
    );
    expect(container.querySelector(NATIVE_CONTROLS)).toBeNull();
  });

  // The native controls leave the title bar in fullscreen, so their place goes too.
  it("gives up the native controls' place in fullscreen", async () => {
    mocks.engine = "tauri";
    const { container, store } = await renderWithConsole(
      <Window.Controls forceOS="macOS" />,
    );
    act(() => {
      store.dispatch(Drift.setWindowFullscreen({ value: true }));
    });
    expect(container.querySelector(NATIVE_CONTROLS)).toBeNull();
  });
});
