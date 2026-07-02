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

const mocks = vi.hoisted((): { engine: "web" | "tauri"; version: string } => ({
  engine: "web",
  version: "1.5.0",
}));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    get ENGINE() {
      return mocks.engine;
    },
  };
});

vi.mock("@tauri-apps/api/app", () => ({
  getVersion: vi.fn(async () => mocks.version),
}));

vi.mock("@tauri-apps/plugin-updater", () => ({ check: vi.fn(async () => null) }));

import { renderWithModals } from "@/platform/modals/testutil";
import { Version } from "@/platform/version";

describe("version Badge", () => {
  beforeEach(() => {
    mocks.engine = "web";
    mocks.version = "1.5.0";
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("should render a version-labelled button", () => {
    renderWithModals(<Version.Badge />);
    const button = screen.getByRole("button");
    expect((button.textContent ?? "").startsWith("v")).toBe(true);
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
