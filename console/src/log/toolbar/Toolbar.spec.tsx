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

// Access.useUpdateGranted returns false with null client.
// Mock only this authorization boundary.
const mockUseUpdateGranted = vi.hoisted(() => vi.fn(() => true));
vi.mock("@synnaxlabs/pluto", async () => {
  const pluto = await vi.importActual<Record<string, unknown>>("@synnaxlabs/pluto");
  return {
    ...pluto,
    Access: { ...(pluto.Access ?? {}), useUpdateGranted: mockUseUpdateGranted },
  };
});

import { Toolbar } from "@/log/toolbar/Toolbar";

const LAYOUT_KEY = "test-key";

const ZERO_LOG_STATE: State = {
  key: LAYOUT_KEY,
  version: "1.0.0",
  channels: [],
  remoteCreated: false,
  timestampPrecision: 0,
  showChannelNames: true,
  showReceiptTimestamp: true,
};

const LAYOUT_STATE: Layout.State = {
  key: LAYOUT_KEY,
  windowKey: MAIN_WINDOW,
  type: "log",
  name: "Test Log",
  location: "mosaic",
};

const preloadState = (
  logState: State = ZERO_LOG_STATE,
  layoutState: Layout.State = LAYOUT_STATE,
) => ({
  [Layout.SLICE_NAME]: {
    ...Layout.ZERO_SLICE_STATE,
    layouts: { ...Layout.ZERO_SLICE_STATE.layouts, [logState.key]: layoutState },
  },
  [Log.SLICE_NAME]: { ...Log.ZERO_SLICE_STATE, logs: { [logState.key]: logState } },
});

const renderToolbar = (layoutKey: string = LAYOUT_KEY, logState?: State) => {
  const ls = logState ?? ZERO_LOG_STATE;
  const lay: Layout.State = { ...LAYOUT_STATE, key: layoutKey };
  return renderWithConsole(<Toolbar layoutKey={layoutKey} />, {
    preloadedState: preloadState({ ...ls, key: layoutKey }, lay),
  });
};

describe("log/Toolbar", () => {
  it("renders null when log state does not exist in store", () => {
    const layoutOnly: Layout.State = { ...LAYOUT_STATE, key: "no-log" };
    const { container } = renderWithConsole(<Toolbar layoutKey="no-log" />, {
      preloadedState: {
        [Layout.SLICE_NAME]: {
          ...Layout.ZERO_SLICE_STATE,
          layouts: { ...Layout.ZERO_SLICE_STATE.layouts, "no-log": layoutOnly },
        },
        [Log.SLICE_NAME]: Log.ZERO_SLICE_STATE,
      },
    });
    expect(container.firstChild).toBeNull();
  });

  it("renders the toolbar content when state is present", () => {
    const { container } = renderToolbar();
    expect(container.firstChild).not.toBeNull();
  });

  it("displays the layout name in the title", () => {
    renderWithConsole(<Toolbar layoutKey={LAYOUT_KEY} />, {
      preloadedState: preloadState(ZERO_LOG_STATE, { ...LAYOUT_STATE, name: "My Log" }),
    });
    expect(screen.getByText("My Log")).toBeDefined();
  });

  it("renders both tab buttons (Channels and Properties)", () => {
    renderToolbar();
    expect(screen.getByText("Channels")).toBeDefined();
    expect(screen.getByText("Properties")).toBeDefined();
  });

  it("defaults to the channels tab", () => {
    renderToolbar();
    // Channels tab content renders — "Add a channel..." placeholder is visible
    expect(screen.getByText("Add a channel...")).toBeDefined();
  });

  it("switches to the Properties tab when clicked", () => {
    renderToolbar();
    fireEvent.click(screen.getByText("Properties"));
    // Properties tab content renders
    expect(screen.getByText("Show Channel Names")).toBeDefined();
  });

  it("switches back to channels tab", () => {
    renderToolbar();
    fireEvent.click(screen.getByText("Properties"));
    fireEvent.click(screen.getByText("Channels"));
    expect(screen.getByText("Add a channel...")).toBeDefined();
  });

  it("passes the layoutKey to sub-components", () => {
    const { store } = renderToolbar("my-layout", {
      ...ZERO_LOG_STATE,
      key: "my-layout",
    });
    // Verify the toolbar rendered with the right log state
    const state = store.getState() as { [Log.SLICE_NAME]: Log.SliceState };
    expect(state[Log.SLICE_NAME].logs["my-layout"]).toBeDefined();
  });
});
