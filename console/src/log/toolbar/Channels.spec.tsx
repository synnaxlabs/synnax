// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Layout } from "@/layout";
import { Log } from "@/log";
import { type State } from "@/log/types/v1";
import { renderWithConsole } from "@/testUtils";

// Access.useUpdateGranted returns false with null client.
// Mock only this authorization boundary so controls are enabled for tests.
const mockUseUpdateGranted = vi.hoisted(() => vi.fn(() => true));
vi.mock("@synnaxlabs/pluto", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@synnaxlabs/pluto")>();
  return {
    ...actual,
    Access: { ...actual.Access, useUpdateGranted: mockUseUpdateGranted },
  };
});

import { Channels } from "@/log/toolbar/Channels";

const ZERO_STATE: State = {
  key: "test-key",
  version: "1.0.0",
  channels: [],
  remoteCreated: false,
  timestampPrecision: 0,
  showChannelNames: true,
  showReceiptTimestamp: true,
};

const LAYOUT_STATE: Layout.State = {
  key: "test-key",
  windowKey: MAIN_WINDOW,
  type: "log",
  name: "Test Log",
  location: "mosaic",
};

const preloadState = (logState: State = ZERO_STATE) => ({
  [Layout.SLICE_NAME]: {
    ...Layout.ZERO_SLICE_STATE,
    layouts: { ...Layout.ZERO_SLICE_STATE.layouts, [logState.key]: LAYOUT_STATE },
  },
  [Log.SLICE_NAME]: { ...Log.ZERO_SLICE_STATE, logs: { [logState.key]: logState } },
});

describe("log/toolbar/Channels", () => {
  it("renders null when state is null", () => {
    const { container } = renderWithConsole(<Channels layoutKey="nonexistent" />, {
      preloadedState: preloadState(ZERO_STATE),
    });
    expect(container.firstChild).toBeNull();
  });

  it("renders the add-channel row when state is present and channels are empty", () => {
    renderWithConsole(<Channels layoutKey="test-key" />, {
      preloadedState: preloadState(ZERO_STATE),
    });
    expect(screen.getByText("Add a channel...")).toBeDefined();
  });

  it("renders a row for each active channel plus the add row", () => {
    const { container } = renderWithConsole(<Channels layoutKey="test-key" />, {
      preloadedState: preloadState({
        ...ZERO_STATE,
        channels: [
          { channel: 10, color: "", notation: "standard", precision: -1, alias: "" },
          { channel: 20, color: "", notation: "standard", precision: -1, alias: "" },
        ],
      }),
    });
    const rows = container.querySelectorAll(".console-log__channel-row");
    // 2 channel rows + 1 add row
    expect(rows.length).toBe(3);
  });

  it("always renders the add-channel row at the bottom", () => {
    renderWithConsole(<Channels layoutKey="test-key" />, {
      preloadedState: preloadState({
        ...ZERO_STATE,
        channels: [
          { channel: 10, color: "", notation: "standard", precision: -1, alias: "" },
          { channel: 20, color: "", notation: "standard", precision: -1, alias: "" },
        ],
      }),
    });
    expect(screen.getByText("Add a channel...")).toBeDefined();
  });

  it("renders a remove button per channel row", () => {
    const { container } = renderWithConsole(<Channels layoutKey="test-key" />, {
      preloadedState: preloadState({
        ...ZERO_STATE,
        channels: [
          { channel: 10, color: "", notation: "standard", precision: -1, alias: "" },
        ],
      }),
    });
    const removeButtons = container.querySelectorAll(
      ".console-log__channel-row button:not([disabled])",
    );
    // Each channel row has a remove button; add row button is disabled
    expect(removeButtons.length).toBeGreaterThanOrEqual(1);
  });

  it("disables controls when user lacks edit permission", () => {
    mockUseUpdateGranted.mockReturnValue(false);
    const { container } = renderWithConsole(<Channels layoutKey="test-key" />, {
      preloadedState: preloadState({
        ...ZERO_STATE,
        channels: [
          { channel: 10, color: "", notation: "standard", precision: -1, alias: "" },
        ],
      }),
    });
    // Checkboxes (Input.Switch) and inputs should be disabled
    const checkboxes = container.querySelectorAll('input[type="checkbox"]');
    for (const cb of checkboxes)
      expect((cb as HTMLInputElement).disabled).toBe(true);
    mockUseUpdateGranted.mockReturnValue(true);
  });
});
