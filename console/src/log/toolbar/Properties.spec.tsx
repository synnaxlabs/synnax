// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Layout } from "@/layout";
import { Log } from "@/log";
import { type State } from "@/log/types/v1";
import { renderWithConsole } from "@/testUtils";

// Access.useUpdateGranted returns false with null client (no permissions).
// Mock only this authorization boundary so controls are enabled for interaction tests.
const mockUseUpdateGranted = vi.hoisted(() => vi.fn(() => true));
vi.mock("@synnaxlabs/pluto", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@synnaxlabs/pluto")>();
  return {
    ...actual,
    Access: { ...actual.Access, useUpdateGranted: mockUseUpdateGranted },
  };
});

import { Properties } from "@/log/toolbar/Properties";

const LOG_STATE: State = {
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

const preloadState = (logState: State = LOG_STATE) => ({
  [Layout.SLICE_NAME]: {
    ...Layout.ZERO_SLICE_STATE,
    layouts: { ...Layout.ZERO_SLICE_STATE.layouts, [logState.key]: LAYOUT_STATE },
  },
  [Log.SLICE_NAME]: { ...Log.ZERO_SLICE_STATE, logs: { [logState.key]: logState } },
});

describe("log/toolbar/Properties", () => {
  it("renders null when state is null", () => {
    const { container } = renderWithConsole(<Properties layoutKey="nonexistent" />, {
      preloadedState: preloadState(LOG_STATE),
    });
    expect(container.firstChild).toBeNull();
  });

  it("renders timestamp precision and show channel names controls", () => {
    renderWithConsole(<Properties layoutKey="test-key" />, {
      preloadedState: preloadState(LOG_STATE),
    });
    expect(screen.getByText("Receipt Timestamp Precision")).toBeDefined();
    expect(screen.getByText("Show Channel Names")).toBeDefined();
  });

  it("renders the current showChannelNames value", () => {
    const { container } = renderWithConsole(<Properties layoutKey="test-key" />, {
      preloadedState: preloadState({ ...LOG_STATE, showChannelNames: false }),
    });
    const checkboxes = container.querySelectorAll('input[type="checkbox"]');
    const showChannelNamesCheckbox = checkboxes[checkboxes.length - 1];
    expect((showChannelNamesCheckbox as HTMLInputElement).checked).toBe(false);
  });

  it("renders the current showReceiptTimestamp value", () => {
    const { container } = renderWithConsole(<Properties layoutKey="test-key" />, {
      preloadedState: preloadState({ ...LOG_STATE, showReceiptTimestamp: false }),
    });
    const checkboxes = container.querySelectorAll('input[type="checkbox"]');
    const showReceiptTimestampCheckbox = checkboxes[0];
    expect((showReceiptTimestampCheckbox as HTMLInputElement).checked).toBe(false);
  });

  it("updates store when showChannelNames toggle is clicked", () => {
    const { store, container } = renderWithConsole(
      <Properties layoutKey="test-key" />,
      { preloadedState: preloadState(LOG_STATE) },
    );
    const checkboxes = container.querySelectorAll('input[type="checkbox"]');
    const showChannelNamesCheckbox = checkboxes[checkboxes.length - 1];
    fireEvent.click(showChannelNamesCheckbox);
    const state = store.getState() as { [Log.SLICE_NAME]: Log.SliceState };
    expect(state[Log.SLICE_NAME].logs["test-key"].showChannelNames).toBe(false);
  });

  it("updates store when showReceiptTimestamp toggle is clicked", () => {
    const { store, container } = renderWithConsole(
      <Properties layoutKey="test-key" />,
      { preloadedState: preloadState(LOG_STATE) },
    );
    const checkboxes = container.querySelectorAll('input[type="checkbox"]');
    const showReceiptTimestampCheckbox = checkboxes[0];
    fireEvent.click(showReceiptTimestampCheckbox);
    const state = store.getState() as { [Log.SLICE_NAME]: Log.SliceState };
    expect(state[Log.SLICE_NAME].logs["test-key"].showReceiptTimestamp).toBe(false);
  });

  it("disables controls when user lacks edit permission", () => {
    mockUseUpdateGranted.mockReturnValue(false);
    const { container } = renderWithConsole(<Properties layoutKey="test-key" />, {
      preloadedState: preloadState(LOG_STATE),
    });
    const checkboxes = container.querySelectorAll('input[type="checkbox"]');
    for (const cb of checkboxes)
      expect((cb as HTMLInputElement).disabled).toBe(true);
    mockUseUpdateGranted.mockReturnValue(true);
  });
});
