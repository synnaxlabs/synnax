// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, screen } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Log } from "@/log/Log";
import { createSynnaxWrapper } from "@/testutil/Synnax";
import { Triggers } from "@/triggers";

// Partial Aether mock: only intercepts type "log" for controlling worker-computed
// state (empty, scrolling, selectionStart, etc.). All other Aether consumers
// (Status, Flux, etc.) get the real implementation via the test providers.
// Type assertions below follow existing vi.mock patterns (vitest doesn't expose
// module types from importOriginal without import() annotations, which lint forbids).
const mockAetherUse = vi.hoisted(() => vi.fn());
vi.mock("@/aether", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  const realAether = actual.Aether as Record<string, unknown>;
  return {
    ...actual,
    Aether: {
      ...realAether,
      use: (...args: unknown[]) => {
        const opts = args[0] as { type?: string } | undefined;
        if (opts?.type === "log") return mockAetherUse(...args);
        return (realAether.use as Function)(...args);
      },
    },
  };
});

const SynnaxWrapper = createSynnaxWrapper({ client: null });
const Wrapper: FC<PropsWithChildren> = ({ children }) => (
  <SynnaxWrapper>
    <Triggers.Provider>{children}</Triggers.Provider>
  </SynnaxWrapper>
);

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

const renderLog = (props: Record<string, unknown> = {}) =>
  render(<Log {...props} />, { wrapper: Wrapper });

const getLogDiv = (container: HTMLElement): HTMLElement => {
  const div = container.querySelector(".pluto-log");
  if (div == null) throw new Error(".pluto-log not found");
  return div as HTMLElement;
};

const getAetherInitialState = (): Record<string, unknown> =>
  mockAetherUse.mock.calls[0][0].initialState;

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
      renderLog();
      expect(screen.getByText("Empty Log")).toBeDefined();
    });

    it("should render the live button when not empty", () => {
      setupAether({ empty: false, entryCount: 10 });
      const { container } = renderLog();
      const liveButton = container.querySelector(".pluto-log__live");
      expect(liveButton).not.toBeNull();
    });

    it("should render custom empty content when provided", () => {
      renderLog({ emptyContent: <div data-testid="custom-empty">No data</div> });
      expect(screen.getByTestId("custom-empty")).toBeDefined();
    });

    it("should apply className to the container div", () => {
      const { container } = renderLog({ className: "my-class" });
      const div = container.querySelector(".my-class");
      expect(div).not.toBeNull();
    });
  });

  describe("live button", () => {
    it("should toggle scrolling when clicked", () => {
      const { setState } = setupAether({ empty: false, scrolling: false });
      const { container } = renderLog();
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
      const { container } = renderLog();
      const logDiv = getLogDiv(container);
      fireEvent.mouseDown(logDiv, { button: 0, clientY: 50 });
      expect(setState).toHaveBeenCalled();
    });

    it("should ignore non-left mouse button on mouse down", () => {
      const { setState } = setupAether({
        empty: false,
        computedLineHeight: 16,
      });
      const { container } = renderLog();
      const logDiv = getLogDiv(container);
      const callsBefore = setState.mock.calls.length;
      fireEvent.mouseDown(logDiv, { button: 2, clientY: 50 });
      expect(setState.mock.calls.length).toBe(callsBefore);
    });

    it("should handle mouse move events", () => {
      const { setState } = setupAether({
        empty: false,
        computedLineHeight: 16,
      });
      const { container } = renderLog();
      const logDiv = getLogDiv(container);
      fireEvent.mouseDown(logDiv, { button: 0, clientY: 50 });
      const callsAfterDown = setState.mock.calls.length;
      fireEvent.mouseMove(logDiv, { clientY: 80 });
      expect(setState.mock.calls.length).toBeGreaterThan(callsAfterDown);
    });

    it("should handle mouse up events", () => {
      setupAether({ empty: false, computedLineHeight: 16 });
      const { container } = renderLog();
      const logDiv = getLogDiv(container);
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
      const { container } = renderLog();
      const logDiv = getLogDiv(container);
      fireEvent.mouseDown(logDiv, { button: 0, clientY: 100, shiftKey: true });
      expect(setState).toHaveBeenCalled();
    });
  });

  describe("wheel events", () => {
    it("should call setState on scroll up", () => {
      const { setState } = setupAether({ empty: false });
      const { container } = renderLog();
      const logDiv = getLogDiv(container);
      fireEvent.wheel(logDiv, { deltaY: -100 });
      expect(setState).toHaveBeenCalled();
    });

    it("should call setState on scroll down", () => {
      const { setState } = setupAether({ empty: false, scrolling: false });
      const { container } = renderLog();
      const logDiv = getLogDiv(container);
      fireEvent.wheel(logDiv, { deltaY: 100 });
      expect(setState).toHaveBeenCalled();
    });
  });

  describe("copy", () => {
    it("should no-op onCopy when no text is selected", () => {
      setupAether({ empty: false, selectedText: "" });
      const { container } = renderLog();
      const logDiv = getLogDiv(container);
      const prevented = fireEvent.copy(logDiv);
      expect(prevented).toBe(true);
    });

    it("should write to clipboardData onCopy when text is selected", () => {
      setupAether({
        empty: false,
        selectedText: "hello",
        selectedLines: [{ text: "hello", color: "" }],
      });
      const { container } = renderLog();
      const logDiv = getLogDiv(container);
      const setData = vi.fn();
      const event = new Event("copy", { bubbles: true });
      Object.defineProperty(event, "clipboardData", {
        value: { setData },
      });
      logDiv.dispatchEvent(event);
      expect(setData).toHaveBeenCalledWith("text/plain", "hello");
      expect(setData).toHaveBeenCalledWith(
        "text/html",
        expect.stringContaining("hello"),
      );
    });
  });

  describe("channel name resolution", () => {
    it("should pass channels to aether state", () => {
      renderLog({
        channels: [{ channel: 1 }, { channel: 2 }, { channel: "virtual" }],
      });
      expect(mockAetherUse).toHaveBeenCalled();
      expect(getAetherInitialState().channels).toEqual([
        { channel: 1 },
        { channel: 2 },
        { channel: "virtual" },
      ]);
    });
  });

  describe("props forwarding", () => {
    it("should pass showChannelNames to aether state", () => {
      renderLog({ showChannelNames: false });
      expect(getAetherInitialState().showChannelNames).toBe(false);
    });

    it("should pass timestampPrecision to aether state", () => {
      renderLog({ timestampPrecision: 3 });
      expect(getAetherInitialState().timestampPrecision).toBe(3);
    });

    it("should default visible to true", () => {
      renderLog();
      expect(getAetherInitialState().visible).toBe(true);
    });

    it("should pass channels with configs to aether state", () => {
      const channels = [{ channel: 1, color: "#ff0000" }, { channel: 2 }];
      renderLog({ channels });
      expect(getAetherInitialState().channels).toEqual(channels);
    });
  });

  describe("mouseYToEntryIndex", () => {
    it("should handle mouse down when computedLineHeight is 0", () => {
      const { setState } = setupAether({
        empty: false,
        computedLineHeight: 0,
        visibleStart: 0,
      });
      const { container } = renderLog();
      const logDiv = getLogDiv(container);
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
      const { container } = renderLog();
      const logDiv = getLogDiv(container);
      fireEvent.mouseDown(logDiv, { button: 0, clientY: 150 });
      expect(setState).toHaveBeenCalled();
    });
  });
});
