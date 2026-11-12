// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
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

import { Modals } from "@/platform/modals";
import { Version } from "@/platform/version";
import {
  callsOf,
  clearUpdater,
  createStatus,
  createWrapper,
  mockLatestCore,
  mockUpdater,
  renderProvider,
} from "@/platform/version/testutil";

const UPDATE = { name: "Update and restart" };

describe("Version.Update", () => {
  let latestCore: MockInstance<typeof fetch>;
  beforeEach(() => {
    latestCore = mockLatestCore("1.5.0");
  });

  afterEach(async () => {
    await clearUpdater();
    latestCore.mockRestore();
  });

  it("should share one download across every view of the update", async () => {
    const ipc = mockUpdater({
      served: "1.5.1",
      download: async () => await new Promise<void>(() => {}),
    });
    render(
      <>
        <Version.Update />
        <Version.Update />
      </>,
      { wrapper: await createWrapper() },
    );
    await waitFor(() => expect(screen.getAllByRole("button", UPDATE)).toHaveLength(2));
    await act(async () => {
      fireEvent.click(screen.getAllByRole("button", UPDATE)[0]);
    });
    await waitFor(() =>
      expect(screen.getAllByText("Downloading update")).toHaveLength(2),
    );
    expect(screen.queryByRole("button", UPDATE)).toBeNull();
    expect(callsOf(ipc, "plugin:updater|download")).toHaveLength(1);
  });

  it("should offer the update again after it fails", async () => {
    const ipc = mockUpdater({
      served: "1.5.1",
      download: async () => {
        throw new Error("download failed");
      },
    });
    render(<Version.Update />, { wrapper: await createWrapper() });
    fireEvent.click(await screen.findByRole("button", UPDATE));
    expect(await screen.findByText("Failed to update Console")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", UPDATE));
    await waitFor(() =>
      expect(callsOf(ipc, "plugin:updater|download")).toHaveLength(2),
    );
  });

  it("should clear a failed update when the checks start again", async () => {
    mockUpdater({
      served: "1.5.1",
      download: async () => {
        throw new Error("download failed");
      },
    });
    const setStatus = await renderProvider(
      <>
        <Version.Update />
        <Modals.Stack />
      </>,
    );
    fireEvent.click(await screen.findByRole("button", UPDATE));
    expect(await screen.findByText("Failed to update Console")).toBeTruthy();
    setStatus(createStatus("1.5.1"));
    expect(await screen.findByText("Version 1.5.1 available")).toBeTruthy();
    expect(screen.queryByText("Failed to update Console")).toBeNull();
  });
});
