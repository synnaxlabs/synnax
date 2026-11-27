// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, describe, expect, it, vi } from "vitest";

import type { Runtime } from "@/session/runtime";

const tauri = vi.hoisted(() => ({ value: false }));

vi.mock("@tauri-apps/api/core", async (importOriginal) => ({
  ...(await importOriginal<Record<string, unknown>>()),
  isTauri: () => tauri.value,
}));

const load = async (): Promise<typeof Runtime> => {
  vi.resetModules();
  return (await import("@/session/runtime")).Runtime;
};

describe("Runtime.CORE_SELECTABLE", () => {
  afterEach(() => {
    tauri.value = false;
    vi.unstubAllGlobals();
  });

  it("should be false for a web build that a Core serves", async () => {
    vi.stubGlobal("IS_DEV", false);
    expect((await load()).CORE_SELECTABLE).toBe(false);
  });

  it("should be true on the dev server", async () => {
    vi.stubGlobal("IS_DEV", true);
    expect((await load()).CORE_SELECTABLE).toBe(true);
  });

  it("should be true on desktop", async () => {
    vi.stubGlobal("IS_DEV", false);
    tauri.value = true;
    expect((await load()).CORE_SELECTABLE).toBe(true);
  });
});
