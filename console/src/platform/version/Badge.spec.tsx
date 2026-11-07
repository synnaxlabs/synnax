// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(
  (): { engine: "web" | "tauri"; version: string; update: unknown } => ({
    engine: "web",
    version: "1.5.0",
    update: null,
  }),
);

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

vi.mock("@tauri-apps/api/app", () => ({
  getVersion: vi.fn(async () => mocks.version),
}));

vi.mock("@tauri-apps/plugin-updater", () => ({
  check: vi.fn(async () => mocks.update),
}));

import { check } from "@tauri-apps/plugin-updater";

import { renderWithModals } from "@/platform/modals/testutil";
import { Version } from "@/platform/version";

const checkMock = vi.mocked(check);

describe("version Badge", () => {
  beforeEach(() => {
    mocks.engine = "web";
    mocks.version = "1.5.0";
    mocks.update = null;
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("should render nothing while the Console is current", async () => {
    mocks.engine = "tauri";
    renderWithModals(<Version.Badge />);
    await waitFor(() => expect(checkMock).toHaveBeenCalled());
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("should prompt to update when an update is available", async () => {
    mocks.engine = "tauri";
    mocks.update = { version: "9.9.9" };
    renderWithModals(<Version.Badge />);
    expect(await screen.findByText("Update available")).toBeTruthy();
  });

  it("should open the info modal when the update prompt is clicked", async () => {
    mocks.engine = "tauri";
    mocks.version = "2.4.6";
    mocks.update = { version: "9.9.9" };
    renderWithModals(<Version.Badge />);
    fireEvent.click(await screen.findByText("Update available"));
    await waitFor(() => expect(screen.getByText("Console v2.4.6")).toBeTruthy());
  });
});
