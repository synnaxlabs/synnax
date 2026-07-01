// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri"; label: string } => ({
  engine: "web",
  label: "main",
}));

vi.mock("@/session/runtime/runtime", () => ({
  get ENGINE() {
    return mocks.engine;
  },
}));

vi.mock("@tauri-apps/api/window", () => ({
  getCurrentWindow: () => ({ label: mocks.label }),
}));

import { Runtime } from "@/session/runtime";

describe("Runtime.isMainWindow", () => {
  beforeEach(() => {
    mocks.engine = "web";
    mocks.label = "main";
  });

  it("should always report true in the web engine", () => {
    mocks.engine = "web";
    mocks.label = "some-child-window";
    expect(Runtime.isMainWindow()).toBe(true);
  });

  it("should report true in tauri when the current window is the main window", () => {
    mocks.engine = "tauri";
    mocks.label = "main";
    expect(Runtime.isMainWindow()).toBe(true);
  });

  it("should report false in tauri from a non-main window", () => {
    mocks.engine = "tauri";
    mocks.label = "child-window";
    expect(Runtime.isMainWindow()).toBe(false);
  });
});
