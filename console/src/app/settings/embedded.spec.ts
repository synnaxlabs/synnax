// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri"; coreEmbedded?: boolean } => ({
  engine: "tauri",
  coreEmbedded: true,
}));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { renderSettings } from "@/app/settings/testutil";

describe("Settings with an embedded Core", () => {
  it("should hold only usage data in the Account group and hide Users", async () => {
    await renderSettings();
    expect(await screen.findByRole("menuitem", { name: "Usage data" })).toBeTruthy();
    expect(await screen.findByRole("menuitem", { name: "Labels" })).toBeTruthy();
    expect(screen.queryByRole("menuitem", { name: "Profile" })).toBeNull();
    expect(screen.queryByRole("menuitem", { name: "API keys" })).toBeNull();
    expect(screen.queryByRole("menuitem", { name: "Users" })).toBeNull();
  });
});
