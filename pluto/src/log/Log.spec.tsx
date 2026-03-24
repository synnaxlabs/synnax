// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// Aether.use mock is essential: it controls worker-computed state (empty, scrolling,
// selectionStart, etc.) that cannot be set via props. This is the ONLY module mock.
const mockAetherUse = vi.hoisted(() => vi.fn());
vi.mock("@/aether", () => ({
  Aether: { use: (...args: unknown[]) => mockAetherUse(...args) },
}));

// Channel.useRetrieveMultiple needs Flux provider (not available without Aether).
// Canvas.useRegion needs Aether. Triggers.use needs Triggers.Provider.
// These hooks depend on providers that require real Aether internally,
// so they must remain mocked while Aether.use is mocked.
const mockUseRetrieveMultiple = vi.hoisted(() =>
  vi.fn(
    (_arg?: unknown) =>
      ({ data: null }) as { data: Array<{ key: number; name: string }> | null },
  ),
);
vi.mock("@/channel", () => ({
  Channel: {
    useRetrieveMultiple: (...args: unknown[]) => mockUseRetrieveMultiple(...args),
  },
}));
vi.mock("@/vis/canvas", () => ({
  Canvas: { useRegion: () => vi.fn() },
}));
vi.mock("@/triggers", () => ({ Triggers: { use: vi.fn() } }));

// All UI components (Button, Icon, Menu, Status, CSS) render for real.
import { Log } from "@/log/Log";

const DEFAULT_STATE = {
  region: { one: { x: 0, y: 0 }, two: { x: 400, y: 500 } },
  wheelPos: 0,
  scrolling: false,
  empty: true,
  visible: true,
  showChannelNames: true,
  timestampPrecision: 0,
  channelNames: {},
  channels: [],
  telem: { type: "noop-log-source", props: {}, variant: "source", valueType: "log" },
  font: "p",
  color: { r: 0, g: 0, b: 0, a: 0 },
  overshoot: { x: 0, y: 0 },
  selectionStart: -1,
  selectionEnd: -1,
  visibleStart: 0,
  selectedText: "",
  selectedLines: [],
  computedLineHeight: 16,
  entryCount: 0,
  copyFlash: false,
};

const setupAether = (overrides: Record<string, unknown> = {}) => {
  const setState = vi.fn();
  const state = { ...DEFAULT_STATE, ...overrides };
  mockAetherUse.mockReturnValue(["test-key", state, setState]);
  return { setState, state };
};

describe("log/Log", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupAether();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("rendering", () => {
    it("should render the empty content when state is empty", () => {
      render(<Log />);
      expect(screen.getByText("Empty Log")).toBeDefined();
    });

    it("should render the live button when not empty", () => {
      setupAether({ empty: false, entryCount: 10 });
      const { container } = render(<Log />);
      const liveButton = container.querySelector(".pluto-log__live");
      expect(liveButton).not.toBeNull();
    });

    it("should render custom empty content when provided", () => {
      render(<Log emptyContent={<div data-testid="custom-empty">No data</div>} />);
      expect(screen.getByTestId("custom-empty")).toBeDefined();
    });

    it("should apply className to the container div", () => {
      const { container } = render(<Log className="my-class" />);
      const div = container.querySelector(".my-class");
      expect(div).not.toBeNull();
    });
  });

  describe("live button", () => {
    it("should toggle scrolling when clicked", () => {
      const { setState } = setupAether({ empty: false, scrolling: false });
      const { container } = render(<Log />);
      const btn = container.querySelector(".pluto-log__live") as HTMLElement;
      fireEvent.click(btn);
      expect(setState).toHaveBeenCalled();
    });
  });

  describe("mouse interactions", () => {
    it("should handle mouse down events on left click", () => {
      const { setState } = setupAether({
        empty: false,
        computedLineHeight: 16,
        visibleStart: 0,
      });
      const { container } = render(<Log />);
      const logDiv = container.querySelector(".pluto-log") as HTMLElement;
      fireEvent.mouseDown(logDiv, { button: 0, clientY: 50 });
      expect(setState).toHaveBeenCalled();
    });

    it("should ignore non-left mouse button on mouse down", () => {
      const { setState } = setupAether({
        empty: false,
        computedLineHeight: 16,
      });
      const { container } = render(<Log />);
      const logDiv = container.querySelector(".pluto-log") as HTMLElement;
      const callsBefore = setState.mock.calls.length;
      fireEvent.mouseDown(logDiv, { button: 2, clientY: 50 });
      expect(setState.mock.calls.length).toBe(callsBefore);
    });

    it("should handle mouse move events", () => {
      const { setState } = setupAether({
        empty: false,
        computedLineHeight: 16,
      });
      const { container } = render(<Log />);
      const logDiv = container.querySelector(".pluto-log") as HTMLElement;
      fireEvent.mouseDown(logDiv, { button: 0, clientY: 50 });
      const callsAfterDown = setState.mock.calls.length;
      fireEvent.mouseMove(logDiv, { clientY: 80 });
      expect(setState.mock.calls.length).toBeGreaterThan(callsAfterDown);
    });

    it("should handle mouse up events", () => {
      setupAether({ empty: false, computedLineHeight: 16 });
      const { container } = render(<Log />);
      const logDiv = container.querySelector(".pluto-log") as HTMLElement;
      fireEvent.mouseUp(logDiv);
    });

    it("should extend selection with shift+click", () => {
      const { setState } = setupAether({
        empty: false,
        computedLineHeight: 16,
        visibleStart: 0,
        selectionStart: 0,
        selectionEnd: 0,
      });
      const { container } = render(<Log />);
      const logDiv = container.querySelector(".pluto-log") as HTMLElement;
      fireEvent.mouseDown(logDiv, { button: 0, clientY: 100, shiftKey: true });
      expect(setState).toHaveBeenCalled();
    });
  });

  describe("wheel events", () => {
    it("should call setState on scroll up", () => {
      const { setState } = setupAether({ empty: false });
      const { container } = render(<Log />);
      const logDiv = container.querySelector(".pluto-log") as HTMLElement;
      fireEvent.wheel(logDiv, { deltaY: -100 });
      expect(setState).toHaveBeenCalled();
    });

    it("should call setState on scroll down", () => {
      const { setState } = setupAether({ empty: false, scrolling: false });
      const { container } = render(<Log />);
      const logDiv = container.querySelector(".pluto-log") as HTMLElement;
      fireEvent.wheel(logDiv, { deltaY: 100 });
      expect(setState).toHaveBeenCalled();
    });
  });

  describe("channel name resolution", () => {
    it("should filter numeric channels for retrieval", () => {
      render(
        <Log channels={[{ channel: 1 }, { channel: 2 }, { channel: "virtual" }]} />,
      );
      expect(mockUseRetrieveMultiple).toHaveBeenCalledWith({ keys: [1, 2] });
    });

    it("should filter out zero channel keys", () => {
      render(<Log channels={[{ channel: 0 }, { channel: 1 }, { channel: 2 }]} />);
      expect(mockUseRetrieveMultiple).toHaveBeenCalledWith({ keys: [1, 2] });
    });

    it("should pass empty array when no numeric channels", () => {
      render(<Log channels={[{ channel: "virtual1" }, { channel: "virtual2" }]} />);
      expect(mockUseRetrieveMultiple).toHaveBeenCalledWith({ keys: [] });
    });

    it("should build channelNames from retrieved channels", () => {
      mockUseRetrieveMultiple.mockReturnValue({
        data: [
          { key: 1, name: "Temperature" },
          { key: 2, name: "Pressure" },
        ],
      });
      render(<Log channels={[{ channel: 1 }, { channel: 2 }]} />);
      expect(mockAetherUse).toHaveBeenCalled();
      const call = mockAetherUse.mock.calls[0][0];
      expect(call.initialState.channelNames).toEqual({
        "1": "Temperature",
        "2": "Pressure",
      });
    });
  });

  describe("props forwarding", () => {
    it("should pass showChannelNames to aether state", () => {
      render(<Log showChannelNames={false} />);
      const call = mockAetherUse.mock.calls[0][0];
      expect(call.initialState.showChannelNames).toBe(false);
    });

    it("should pass timestampPrecision to aether state", () => {
      render(<Log timestampPrecision={3} />);
      const call = mockAetherUse.mock.calls[0][0];
      expect(call.initialState.timestampPrecision).toBe(3);
    });

    it("should default visible to true", () => {
      render(<Log />);
      const call = mockAetherUse.mock.calls[0][0];
      expect(call.initialState.visible).toBe(true);
    });

    it("should pass channels with configs to aether state", () => {
      const channels = [{ channel: 1, color: "#ff0000" }, { channel: 2 }];
      render(<Log channels={channels} />);
      const call = mockAetherUse.mock.calls[0][0];
      expect(call.initialState.channels).toEqual(channels);
    });
  });

  describe("mouseYToEntryIndex", () => {
    it("should handle mouse down when computedLineHeight is 0", () => {
      const { setState } = setupAether({
        empty: false,
        computedLineHeight: 0,
        visibleStart: 0,
      });
      const { container } = render(<Log />);
      const logDiv = container.querySelector(".pluto-log") as HTMLElement;
      fireEvent.mouseDown(logDiv, { button: 0, clientY: 100 });
      expect(setState).toHaveBeenCalled();
    });

    it("should compute entry index from clientY position", () => {
      const { setState } = setupAether({
        empty: false,
        computedLineHeight: 16,
        visibleStart: 5,
        region: { one: { x: 0, y: 100 }, two: { x: 400, y: 600 } },
      });
      const { container } = render(<Log />);
      const logDiv = container.querySelector(".pluto-log") as HTMLElement;
      fireEvent.mouseDown(logDiv, { button: 0, clientY: 150 });
      expect(setState).toHaveBeenCalled();
    });
  });
});
