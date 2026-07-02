// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach, describe, expect, it, vi } from "vitest";

import type * as RuntimeModule from "@/session/runtime/runtime";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));

vi.mock("@/session/runtime/runtime", async (orig) => ({
  ...(await orig<typeof RuntimeModule>()),
  get ENGINE() {
    return mocks.engine;
  },
}));

import { Window } from "@/platform/window";
import { renderWithConsole } from "@/testutil";

describe("Window.Controls", () => {
  beforeEach(() => {
    mocks.engine = "web";
  });

  it("renders nothing outside the tauri engine", async () => {
    mocks.engine = "web";
    const { container } = await renderWithConsole(<Window.Controls forceOS="macOS" />);
    expect(container.querySelector("button")).toBeNull();
  });

  it("renders the OS window controls in the tauri engine", async () => {
    mocks.engine = "tauri";
    const { container } = await renderWithConsole(<Window.Controls forceOS="macOS" />);
    expect(container.querySelector("button")).not.toBeNull();
  });
});
