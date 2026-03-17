// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, screen } from "@testing-library/react";
import React from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// --- Mocks (hoisted by vitest before imports) ---

const mockAetherUse = vi.fn();
const mockUseRegion = vi.fn(() => vi.fn());
const mockUseRetrieveMultiple = vi.fn(
  (_arg?: unknown) =>
    ({ data: null }) as { data: Array<{ key: number; name: string }> | null },
);
const mockUseContextMenu = vi.fn(() => ({
  className: "mock-menu",
  visible: false,
  close: vi.fn(),
  open: vi.fn(),
}));

vi.mock("@/aether", () => ({
  Aether: { use: (a: unknown) => mockAetherUse(a) },
}));

vi.mock("@/vis/canvas", () => ({
  Canvas: { useRegion: () => mockUseRegion() },
}));

vi.mock("@/channel", () => ({
  Channel: {
    useRetrieveMultiple: (arg: unknown) => mockUseRetrieveMultiple(arg),
  },
}));

vi.mock("@/menu", () => ({
  Menu: {
    useContextMenu: () => mockUseContextMenu(),
    ContextMenu: ({
      children,
    }: {
      children: React.ReactNode;
      menu?: () => React.ReactNode;
      className?: string;
    }) => <div data-testid="context-menu">{children}</div>,
    Menu: ({
      children,
      onChange,
    }: {
      children: React.ReactNode;
      level?: string;
      onChange?: (key: string) => void;
    }) => (
      <div data-testid="menu" onClick={() => onChange?.("copy")}>
        {children}
      </div>
    ),
    Item: ({
      children,
      itemKey,
      disabled,
    }: {
      children: React.ReactNode;
      itemKey: string;
      trigger?: unknown;
      triggerIndicator?: boolean;
      disabled?: boolean;
    }) => (
      <div data-testid={`menu-item-${itemKey}`} data-disabled={String(disabled)}>
        {children}
      </div>
    ),
  },
}));

vi.mock("@/memo", () => ({
  useMemoDeepEqual: (v: unknown) => v,
}));

vi.mock("@/log/Log.css", () => ({}));

vi.mock("@/css", () => ({
  CSS: Object.assign((...args: string[]) => args.filter(Boolean).join(" "), {
    B: (name: string) => `pluto--${name}`,
    BE: (block: string, element: string) => `pluto--${block}__${element}`,
    M: (modifier: string) => `pluto--${modifier}`,
  }),
}));

vi.mock("@/button", () => ({
  Button: {
    Button: ({
      children,
      onClick,
      className,
      tooltip,
    }: {
      children: React.ReactNode;
      onClick?: () => void;
      className?: string;
      tooltip?: string;
      variant?: string;
      tooltipLocation?: unknown;
    }) => (
      <button
        data-testid="live-button"
        className={className}
        onClick={onClick}
        title={tooltip}
      >
        {children}
      </button>
    ),
  },
}));

vi.mock("@/icon", () => ({
  Icon: {
    Dynamic: () => <span data-testid="icon-dynamic" />,
    Copy: () => <span data-testid="icon-copy" />,
  },
}));

vi.mock("@/status/base", () => ({
  Status: {
    Summary: ({ children }: { children: React.ReactNode }) => (
      <div data-testid="status-summary">{children}</div>
    ),
  },
}));

vi.mock("@/triggers", () => ({ Triggers: {} }));

vi.mock("@/log/aether", () => ({
  log: {
    Log: { TYPE: "log" },
    logState: { parse: (v: unknown) => v },
  },
}));

// Imports (after mocks)
import { Log } from "@/log/Log";

const DEFAULT_STATE = {
  region: { one: { x: 0, y: 0 }, two: { x: 400, y: 500 } },
  wheelPos: 0,
  scrolling: false,
  empty: true,
  visible: true,
  showChannelNames: true,
  timestampPrecision: 0,
  channelConfigs: {},
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
    // Default: empty state
    setupAether();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("rendering", () => {
    it("should render the empty content when state is empty", () => {
      render(<Log />);
      expect(screen.getByTestId("status-summary")).toBeDefined();
      expect(screen.getByText("Empty Log")).toBeDefined();
    });

    it("should render the live button when not empty", () => {
      setupAether({ empty: false, entryCount: 10 });
      render(<Log />);
      expect(screen.getByTestId("live-button")).toBeDefined();
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
    it("should show 'Pause Scrolling' tooltip when not scrolling", () => {
      setupAether({ empty: false, scrolling: false });
      render(<Log />);
      expect(screen.getByTestId("live-button").getAttribute("title")).toBe(
        "Pause Scrolling",
      );
    });

    it("should show 'Resume Scrolling' tooltip when scrolling", () => {
      setupAether({ empty: false, scrolling: true });
      render(<Log />);
      expect(screen.getByTestId("live-button").getAttribute("title")).toBe(
        "Resume Scrolling",
      );
    });

    it("should toggle scrolling when clicked", () => {
      const { setState } = setupAether({ empty: false, scrolling: false });
      render(<Log />);
      fireEvent.click(screen.getByTestId("live-button"));
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
      render(<Log />);
      const container = screen.getByTestId("context-menu").firstChild as HTMLElement;
      fireEvent.mouseDown(container, { button: 0, clientY: 50 });
      expect(setState).toHaveBeenCalled();
    });

    it("should ignore non-left mouse button on mouse down", () => {
      const { setState } = setupAether({
        empty: false,
        computedLineHeight: 16,
      });
      render(<Log />);
      const container = screen.getByTestId("context-menu").firstChild as HTMLElement;
      const callsBefore = setState.mock.calls.length;
      fireEvent.mouseDown(container, { button: 2, clientY: 50 });
      // No additional setState calls from the mouseDown handler
      expect(setState.mock.calls.length).toBe(callsBefore);
    });

    it("should handle mouse move events", () => {
      const { setState } = setupAether({
        empty: false,
        computedLineHeight: 16,
      });
      render(<Log />);
      const container = screen.getByTestId("context-menu").firstChild as HTMLElement;
      // Start a drag first
      fireEvent.mouseDown(container, { button: 0, clientY: 50 });
      const callsAfterDown = setState.mock.calls.length;
      fireEvent.mouseMove(container, { clientY: 80 });
      expect(setState.mock.calls.length).toBeGreaterThan(callsAfterDown);
    });

    it("should handle mouse up events", () => {
      setupAether({ empty: false, computedLineHeight: 16 });
      render(<Log />);
      const container = screen.getByTestId("context-menu").firstChild as HTMLElement;
      fireEvent.mouseUp(container);
    });

    it("should extend selection with shift+click", () => {
      const { setState } = setupAether({
        empty: false,
        computedLineHeight: 16,
        visibleStart: 0,
        selectionStart: 0,
        selectionEnd: 0,
      });
      render(<Log />);
      const container = screen.getByTestId("context-menu").firstChild as HTMLElement;
      fireEvent.mouseDown(container, { button: 0, clientY: 100, shiftKey: true });
      expect(setState).toHaveBeenCalled();
    });
  });

  describe("wheel events", () => {
    it("should update wheelPos on scroll up", () => {
      const { setState } = setupAether({ empty: false });
      render(<Log />);
      const container = screen.getByTestId("context-menu").firstChild as HTMLElement;
      fireEvent.wheel(container, { deltaY: -100 });
      expect(setState).toHaveBeenCalled();
    });

    it("should call setState on scroll up", () => {
      const { setState } = setupAether({ empty: false, scrolling: false });
      render(<Log />);
      const container = screen.getByTestId("context-menu").firstChild as HTMLElement;
      fireEvent.wheel(container, { deltaY: -100 });
      expect(setState).toHaveBeenCalled();
    });

    it("should call setState on scroll down", () => {
      const { setState } = setupAether({ empty: false, scrolling: false });
      render(<Log />);
      const container = screen.getByTestId("context-menu").firstChild as HTMLElement;
      fireEvent.wheel(container, { deltaY: 100 });
      expect(setState).toHaveBeenCalled();
    });
  });

  describe("keyboard shortcuts", () => {
    it("should clear selection on Escape", () => {
      const { setState } = setupAether({
        selectedText: "some text",
        selectionStart: 0,
        selectionEnd: 5,
      });
      render(<Log />);
      fireEvent.keyDown(window, { key: "Escape" });
      expect(setState).toHaveBeenCalled();
    });

    it("should select all on Cmd+A when entries exist", () => {
      const { setState } = setupAether({ entryCount: 10, empty: false });
      render(<Log />);
      fireEvent.keyDown(window, { key: "a", metaKey: true });
      expect(setState).toHaveBeenCalled();
    });

    it("should select all on Ctrl+A when entries exist", () => {
      const { setState } = setupAether({ entryCount: 10, empty: false });
      render(<Log />);
      fireEvent.keyDown(window, { key: "a", ctrlKey: true });
      expect(setState).toHaveBeenCalled();
    });

    it("should not respond to keyboard events in input elements", () => {
      const { setState } = setupAether({ entryCount: 10, selectedText: "text" });
      render(
        <div>
          <Log />
          <input data-testid="test-input" />
        </div>,
      );
      const callsBefore = setState.mock.calls.length;
      const input = screen.getByTestId("test-input");
      fireEvent.keyDown(input, { key: "a", metaKey: true });
      // setState should not be called from the keyboard handler for input elements
      expect(setState.mock.calls.length).toBe(callsBefore);
    });

    it("should not respond to plain key presses without modifier", () => {
      const { setState } = setupAether({ entryCount: 10, selectedText: "text" });
      render(<Log />);
      const callsBefore = setState.mock.calls.length;
      fireEvent.keyDown(window, { key: "c" });
      // Without meta/ctrl, should not trigger copy
      expect(setState.mock.calls.length).toBe(callsBefore);
    });
  });

  describe("channel name resolution", () => {
    it("should filter numeric channels for retrieval", () => {
      render(<Log channels={[1, 2, "virtual"]} />);
      expect(mockUseRetrieveMultiple).toHaveBeenCalledWith({ keys: [1, 2] });
    });

    it("should filter out zero channel keys", () => {
      render(<Log channels={[0, 1, 2]} />);
      expect(mockUseRetrieveMultiple).toHaveBeenCalledWith({ keys: [1, 2] });
    });

    it("should pass empty array when no numeric channels", () => {
      render(<Log channels={["virtual1", "virtual2"]} />);
      expect(mockUseRetrieveMultiple).toHaveBeenCalledWith({ keys: [] });
    });

    it("should build channelNames from retrieved channels", () => {
      mockUseRetrieveMultiple.mockReturnValue({
        data: [
          { key: 1, name: "Temperature" },
          { key: 2, name: "Pressure" },
        ],
      });
      render(<Log channels={[1, 2]} />);
      // Aether.use should be called with channelNames in initialState
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

    it("should pass channelConfigs to aether state", () => {
      const configs = { "1": { color: "#ff0000" } };
      render(<Log channelConfigs={configs} />);
      const call = mockAetherUse.mock.calls[0][0];
      expect(call.initialState.channelConfigs).toEqual(configs);
    });

    it("should pass channels to aether state", () => {
      render(<Log channels={[1, 2, 3]} />);
      const call = mockAetherUse.mock.calls[0][0];
      expect(call.initialState.channels).toEqual([1, 2, 3]);
    });
  });

  describe("context menu", () => {
    it("should render the context menu wrapper", () => {
      render(<Log />);
      expect(screen.getByTestId("context-menu")).toBeDefined();
    });
  });

  describe("mouseYToEntryIndex", () => {
    it("should handle mouse down when computedLineHeight is 0", () => {
      const { setState } = setupAether({
        empty: false,
        computedLineHeight: 0,
        visibleStart: 0,
      });
      render(<Log />);
      const container = screen.getByTestId("context-menu").firstChild as HTMLElement;
      fireEvent.mouseDown(container, { button: 0, clientY: 100 });
      expect(setState).toHaveBeenCalled();
    });

    it("should compute entry index from clientY position", () => {
      const { setState } = setupAether({
        empty: false,
        computedLineHeight: 16,
        visibleStart: 5,
        region: { one: { x: 0, y: 100 }, two: { x: 400, y: 600 } },
      });
      render(<Log />);
      const container = screen.getByTestId("context-menu").firstChild as HTMLElement;
      fireEvent.mouseDown(container, { button: 0, clientY: 150 });
      expect(setState).toHaveBeenCalled();
    });
  });
});
