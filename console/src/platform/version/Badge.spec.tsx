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

import { Version } from "@/platform/version";
import { renderWithModals } from "@/platform/modals/testutil";

describe("version Badge", () => {
  beforeEach(() => {
    mocks.engine = "web";
    mocks.version = "1.5.0";
    mocks.update = null;
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("should not open the info modal when clicked in the browser", async () => {
    renderWithModals(<Version.Badge />);
    const button = screen.getByRole("button");
    expect((button.textContent ?? "").startsWith("v")).toBe(true);
    fireEvent.click(button);
    await waitFor(() => expect(screen.queryByText(/Console v/)).toBeNull());
  });

  it("should highlight the badge when an update is available", async () => {
    mocks.engine = "tauri";
    mocks.update = { version: "9.9.9" };
    renderWithModals(<Version.Badge />);
    await waitFor(() =>
      expect(screen.getByRole("button").style.color).toContain("--pluto-secondary-z"),
    );
  });

  it("should render the resolved app version in tauri", async () => {
    mocks.engine = "tauri";
    mocks.version = "2.4.6";
    renderWithModals(<Version.Badge />);
    await waitFor(() =>
      expect(screen.getByRole("button").textContent).toEqual("v2.4.6"),
    );
  });

  it("should open the info modal when clicked in tauri", async () => {
    mocks.engine = "tauri";
    mocks.version = "2.4.6";
    renderWithModals(<Version.Badge />);
    await waitFor(() =>
      expect(screen.getByRole("button").textContent).toEqual("v2.4.6"),
    );
    fireEvent.click(screen.getByRole("button"));
    await waitFor(() => expect(screen.getByText("Console v2.4.6")).toBeTruthy());
  });
});
