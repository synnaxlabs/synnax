// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { mockWindows } from "@tauri-apps/api/mocks";
import { render, screen } from "@testing-library/react";
import {
  afterEach,
  beforeEach,
  describe,
  expect,
  it,
  type MockInstance,
  vi,
} from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "tauri" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Version } from "@/feature/version";
import {
  clearUpdater,
  createWrapper,
  mockLatestCore,
  mockUpdater,
} from "@/platform/version/testutil";

describe("Version.Settings", () => {
  let latestCore: MockInstance<typeof fetch>;
  beforeEach(() => {
    mocks.engine = "tauri";
    mockWindows(Drift.MAIN_WINDOW);
    // The Core notice asks the docs site for the newest Core.
    latestCore = mockLatestCore("1.5.0");
  });

  afterEach(async () => {
    await clearUpdater();
    latestCore.mockRestore();
  });

  it("should offer the update that the Core serves on Desktop", async () => {
    mockUpdater({ served: "1.5.1" });
    render(<Version.Settings />, { wrapper: await createWrapper() });
    expect(await screen.findByText("Version 1.5.1 available")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Update and restart" })).toBeTruthy();
  });

  it("should leave out updates in a secondary window", async () => {
    mockUpdater({ served: "1.5.1" });
    mockWindows("secondary", Drift.MAIN_WINDOW);
    render(<Version.Settings />, { wrapper: await createWrapper() });
    expect(await screen.findByText("v1.5.0")).toBeTruthy();
    expect(screen.queryByText("Updates")).toBeNull();
  });

  it("should leave out updates on the web", async () => {
    mocks.engine = "web";
    render(<Version.Settings />, { wrapper: await createWrapper() });
    expect(await screen.findByText("v1.5.0")).toBeTruthy();
    expect(screen.queryByText("Updates")).toBeNull();
  });
});
