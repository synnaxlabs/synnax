// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "tauri" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { UncappedFeed } from "@/platform/notifications/testutil";
import { Version } from "@/platform/version";
import {
  callsOf,
  clearUpdater,
  createStatus,
  createWrapper,
  mockUpdater,
} from "@/platform/version/testutil";
import { createCore, createCoreState } from "@/session/core/testutil";
import { getIconButton, renderWithStatus } from "@/testutil";

const renderBadge = async (): Promise<void> => {
  render(<Version.Badge />, { wrapper: await createWrapper() });
};

describe("version Badge", () => {
  beforeEach(() => {
    mocks.engine = "tauri";
  });

  afterEach(async () => {
    await clearUpdater();
    vi.clearAllMocks();
  });

  it("should render nothing while the Console is current", async () => {
    const ipc = mockUpdater({ served: "1.5.0" });
    await renderBadge();
    await waitFor(() => expect(callsOf(ipc, "updater_check")).toHaveLength(1));
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("should prompt to update when an update is available", async () => {
    mockUpdater({ served: "1.4.2" });
    await renderBadge();
    expect(await screen.findByText("Update available")).toBeTruthy();
  });

  it("should open the info modal when the update prompt is clicked", async () => {
    mockUpdater({ version: "1.5.0", served: "1.4.2" });
    await renderBadge();
    fireEvent.click(await screen.findByText("Update available"));
    await waitFor(() => expect(screen.getByText("Console v1.5.0")).toBeTruthy());
  });

  it("should check again when the Core reports another version", async () => {
    const before = mockUpdater({ served: "1.5.0" });
    const core = createCore("Local");
    const { setStatus } = await renderWithStatus(
      <Version.Provider>
        <Version.Badge />
      </Version.Provider>,
      {
        client: null,
        status: createStatus("1.5.0"),
        preloadedState: createCoreState([core], core.key),
      },
    );
    await waitFor(() => expect(callsOf(before, "updater_check")).toHaveLength(1));
    expect(screen.queryByText("Update available")).toBeNull();
    mockUpdater({ served: "1.6.0" });
    setStatus(createStatus("1.6.0"));
    expect(await screen.findByText("Update available")).toBeTruthy();
  });

  it("should keep a silenced update silent when a new check finds it again", async () => {
    const ipc = mockUpdater({ served: "1.4.2" });
    const core = createCore("Local");
    const { setStatus } = await renderWithStatus(
      <Version.Provider>
        <UncappedFeed />
      </Version.Provider>,
      {
        client: null,
        status: createStatus("1.5.0"),
        preloadedState: createCoreState([core], core.key),
      },
    );
    await screen.findByText("Update available");
    fireEvent.click(getIconButton(document.body, "close"));
    await waitFor(() => expect(screen.queryByText("Update available")).toBeNull());
    setStatus(createStatus("1.5.1"));
    await waitFor(() => expect(callsOf(ipc, "updater_check")).toHaveLength(2));
    await act(async () => {});
    expect(screen.queryByText("Update available")).toBeNull();
  });
});
