// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { stubClipboardUnavailable, stubCopyCommand } from "@synnaxlabs/lyra/testutil";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { box, color, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { FC, PropsWithChildren, ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Log } from "@/log";
import { log as aetherLog } from "@/log/aether";
import { createAetherSetup, DEFAULT_STATE, initialStateOf } from "@/log/testutil";
import { createSynnaxWrapper } from "@/testutil";

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
        return (realAether.use as (...a: unknown[]) => unknown)(...args);
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

const setupAether = createAetherSetup(mockAetherUse);

const renderLog = (props: Partial<Log.BaseProps> = {}) =>
  render(<Log.Base {...props} />, { wrapper: Wrapper });

const getLogDiv = (container: HTMLElement): HTMLElement => {
  const div = container.querySelector(".pluto-log");
  if (div == null) throw new Error(".pluto-log not found");
  return div as HTMLElement;
};

const getAetherInitialState = (): Record<string, unknown> =>
  initialStateOf(mockAetherUse);

// Simulates a worker-pushed state change, which reaches the main thread through
// onAetherChange alone — local setState never fires it.
const pushAetherState = (overrides: Record<string, unknown>): void => {
  const { calls } = mockAetherUse.mock;
  calls[calls.length - 1][0].onAetherChange({ ...DEFAULT_STATE, ...overrides });
};

describe("log/Base", () => {
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
      expect(screen.getByText("No log entries")).toBeDefined();
    });

    it("should show how far back a filtered log has searched", () => {
      setupAether({ empty: false, searchedTo: "86400000000000" });
      const { rerender } = renderLog({ timeZone: "UTC" });
      const time = new TimeStamp(86400000000000n).toString("dateTime", "UTC");
      expect(screen.getByText(/Searched back to/).textContent).toContain(time);
      setupAether({ empty: false, searchedTo: "" });
      rerender(<Log.Base />);
      expect(screen.queryByText(/Searched back to/)).toBeNull();
    });

    it("should expose the log role", () => {
      renderLog();
      expect(screen.getByRole("log")).toBeDefined();
    });

    it("should show the loading symbol only while loading", () => {
      setupAether({ empty: false, loading: true });
      const { rerender } = renderLog();
      expect(screen.getByLabelText("Loading")).toBeDefined();
      setupAether({ empty: false, loading: false });
      rerender(<Log.Base />);
      expect(screen.queryByLabelText("Loading")).toBeNull();
    });

    it("should show the full loading state while a log has no entries", () => {
      setupAether({ empty: false, loading: true, blank: true });
      const { rerender } = renderLog();
      expect(screen.getByLabelText("Loading entries")).toBeDefined();
      expect(screen.queryByLabelText("Loading")).toBeNull();
      setupAether({ empty: false, loading: true, blank: false });
      rerender(<Log.Base />);
      expect(screen.queryByLabelText("Loading entries")).toBeNull();
      expect(screen.getByLabelText("Loading")).toBeDefined();
    });

    it("should render children when not empty", () => {
      setupAether({ empty: false });
      renderLog({ children: <div data-testid="overlay" /> });
      expect(screen.getByTestId("overlay")).toBeDefined();
    });

    // A filter that matches nothing empties the log, and its controls must stay.
    it("should render children beside the empty content", () => {
      setupAether({ empty: true });
      renderLog({ children: <div data-testid="overlay" /> });
      expect(screen.getByTestId("overlay")).toBeDefined();
      expect(screen.getByText("No log entries")).toBeDefined();
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

  describe("channel colors", () => {
    it("should send Aether the resolved color of each channel", () => {
      setupAether();
      renderLog({
        channels: [
          log.channelEntryZ.parse({
            channel: 1,
            color: { type: "fixed", value: "#ff0000" },
          }),
          log.channelEntryZ.parse({ channel: 2 }),
        ],
      });
      expect(getAetherInitialState().channelColors).toEqual({
        1: color.construct("#ff0000"),
      });
    });
  });

  describe("onLeave", () => {
    it("should report the last position when the log unmounts", () => {
      setupAether({ position: "42" });
      const onLeave = vi.fn();
      const { unmount } = renderLog({ onLeave });
      expect(onLeave).not.toHaveBeenCalled();
      unmount();
      expect(onLeave).toHaveBeenCalledExactlyOnceWith("42");
    });

    it("should start the log at the position prop", () => {
      setupAether();
      renderLog({ position: "42" });
      expect(mockAetherUse.mock.calls[0][0].initialState.position).toBe("42");
    });
  });

  describe("hold", () => {
    const scrollingResults = (
      setState: ReturnType<typeof vi.fn>,
      state: Record<string, unknown>,
    ): unknown[] =>
      setState.mock.calls
        .map(([updater]) => (typeof updater === "function" ? updater(state) : updater))
        .map((r) => (r as Record<string, unknown>).scrolling);

    it("should sync the hold prop into aether scrolling state", () => {
      const { setState, state } = setupAether({ empty: false, scrolling: false });
      renderLog({ hold: true });
      expect(scrollingResults(setState, state)).toContain(true);
    });

    it("should not pause when the hold prop is false", () => {
      const { setState, state } = setupAether({ empty: false, scrolling: false });
      renderLog({ hold: false });
      expect(scrollingResults(setState, state)).not.toContain(true);
    });

    it("should sync a false hold prop into a paused aether state", () => {
      const { setState, state } = setupAether({ empty: false, scrolling: true });
      renderLog({ hold: false });
      expect(scrollingResults(setState, state)).toContain(false);
    });

    it("should leave aether scrolling alone when no hold prop is given", () => {
      const { setState, state } = setupAether({ empty: false, scrolling: true });
      renderLog();
      const results = scrollingResults(setState, state);
      expect(results).not.toContain(undefined);
      expect(results).not.toContain(false);
    });

    it("should write the pause into aether state on the H trigger", () => {
      const { setState, state } = setupAether({ empty: false, scrolling: false });
      renderLog({ enableTriggers: true });
      fireEvent.keyDown(document.body, { code: "KeyH" });
      fireEvent.keyUp(document.body, { code: "KeyH" });
      expect(scrollingResults(setState, state)).toContain(true);
    });

    it("should write the pause into aether state on scroll up", () => {
      const { setState, state } = setupAether({ empty: false, scrolling: false });
      const { container } = renderLog();
      fireEvent.wheel(getLogDiv(container), { deltaY: -100 });
      expect(scrollingResults(setState, state)).toContain(true);
    });

    it("should call onHold on scroll up", () => {
      setupAether({ empty: false, scrolling: false });
      const onHold = vi.fn();
      const { container } = renderLog({ onHold });
      fireEvent.wheel(getLogDiv(container), { deltaY: -100 });
      expect(onHold).toHaveBeenCalledWith(true);
    });

    it("should not call onHold on scroll down", () => {
      setupAether({ empty: false, scrolling: false });
      const onHold = vi.fn();
      const { container } = renderLog({ onHold });
      fireEvent.wheel(getLogDiv(container), { deltaY: 100 });
      expect(onHold).not.toHaveBeenCalled();
    });

    it("should not call onHold on scroll up when already paused", () => {
      setupAether({ empty: false, scrolling: true });
      const onHold = vi.fn();
      const { container } = renderLog({ hold: true, onHold });
      fireEvent.wheel(getLogDiv(container), { deltaY: -100 });
      expect(onHold).not.toHaveBeenCalled();
    });

    it("should call onHold with the toggled value on the H trigger", () => {
      setupAether({ empty: false, scrolling: false });
      const onHold = vi.fn();
      renderLog({ onHold, enableTriggers: true });
      fireEvent.keyDown(document.body, { code: "KeyH" });
      fireEvent.keyUp(document.body, { code: "KeyH" });
      expect(onHold).toHaveBeenCalledWith(true);
    });

    it("should call onHold when the worker resumes on its own", () => {
      setupAether({ empty: false, scrolling: true });
      const onHold = vi.fn();
      renderLog({ hold: true, onHold });
      pushAetherState({ scrolling: false, resumedAt: 1 });
      expect(onHold).toHaveBeenCalledExactlyOnceWith(false);
    });

    it("should not call onHold when the worker echoes the current hold state", () => {
      setupAether({ empty: false, scrolling: true });
      const onHold = vi.fn();
      renderLog({ hold: true, onHold });
      pushAetherState({ scrolling: true });
      expect(onHold).not.toHaveBeenCalled();
    });

    it("should not call onHold when the worker echoes the previous hold state", () => {
      setupAether({ empty: false, scrolling: true });
      const onHold = vi.fn();
      renderLog({ hold: false, onHold });
      pushAetherState({ scrolling: true });
      expect(onHold).not.toHaveBeenCalled();
    });

    it("should not call onHold when a stale push still shows the log live", () => {
      setupAether({ empty: false, scrolling: false });
      const onHold = vi.fn();
      renderLog({ hold: true, onHold });
      pushAetherState({ scrolling: false });
      expect(onHold).not.toHaveBeenCalled();
    });

    it("should call onHold once per resume", () => {
      setupAether({ empty: false, scrolling: true });
      const onHold = vi.fn();
      renderLog({ hold: true, onHold });
      pushAetherState({ scrolling: false, resumedAt: 1 });
      pushAetherState({ scrolling: false, resumedAt: 1 });
      expect(onHold).toHaveBeenCalledExactlyOnceWith(false);
    });

    it("should ignore a resume stamp older than the last one seen", () => {
      setupAether({ empty: false, scrolling: true });
      const onHold = vi.fn();
      renderLog({ hold: true, onHold });
      pushAetherState({ scrolling: false, resumedAt: 2 });
      pushAetherState({ scrolling: false, resumedAt: 1 });
      expect(onHold).toHaveBeenCalledExactlyOnceWith(false);
    });

    it("should not call onHold on a worker push when hold is uncontrolled", () => {
      setupAether({ empty: false, scrolling: true });
      const onHold = vi.fn();
      renderLog({ onHold });
      pushAetherState({ scrolling: false, resumedAt: 1 });
      expect(onHold).not.toHaveBeenCalled();
    });

    it("should not call onHold on the H trigger when enableTriggers returns false", () => {
      setupAether({ empty: false, scrolling: false });
      const onHold = vi.fn();
      renderLog({ onHold, enableTriggers: () => false });
      fireEvent.keyDown(document.body, { code: "KeyH" });
      fireEvent.keyUp(document.body, { code: "KeyH" });
      expect(onHold).not.toHaveBeenCalled();
    });
  });

  describe("mouse interactions", () => {
    type State = Record<string, unknown>;

    // Returns the state after the last update the log made.
    const lastState = (
      setState: ReturnType<typeof vi.fn>,
      from: State = DEFAULT_STATE,
    ): State => {
      const update = setState.mock.calls.at(-1)?.[0];
      return typeof update === "function" ? update(from) : update;
    };

    // Returns the state after every update the log made, in order, as React applies
    // them.
    const applied = (setState: ReturnType<typeof vi.fn>): State =>
      setState.mock.calls.reduce<State>((state, [update]) => {
        if (typeof update === "function") return update(state);
        return { ...state, ...update };
      }, DEFAULT_STATE);

    const WHOLE = { selectionStartChar: -1, selectionEndChar: -1 };

    it("should select the whole line a click lands on", async () => {
      const { setState, hit } = setupAether({ empty: false });
      hit.mockResolvedValue({ entry: 2, char: 5 });
      const { container } = renderLog();
      fireEvent.mouseDown(getLogDiv(container), {
        button: 0,
        clientX: 30,
        clientY: 50,
      });
      await waitFor(() =>
        expect(lastState(setState)).toMatchObject({
          selectionStart: 2,
          selectionEnd: 2,
          ...WHOLE,
        }),
      );
      expect(hit).toHaveBeenCalledWith({ x: 30, y: 50 });
    });

    it("should clear a line that alone is selected when clicked again", async () => {
      const { setState, hit } = setupAether({ empty: false });
      hit.mockResolvedValue({ entry: 2, char: 5 });
      const { container } = renderLog();
      fireEvent.mouseDown(getLogDiv(container), { button: 0 });
      const selected = { selectionStart: 2, selectionEnd: 2, ...WHOLE };
      await waitFor(() =>
        expect(lastState(setState, selected)).toMatchObject({
          selectionStart: -1,
          selectionEnd: -1,
        }),
      );
    });

    it("should ignore a button other than the left", () => {
      const { hit } = setupAether({ empty: false });
      const { container } = renderLog();
      fireEvent.mouseDown(getLogDiv(container), { button: 2 });
      expect(hit).not.toHaveBeenCalled();
    });

    it("should select the characters between a drag's ends", async () => {
      const { setState, hit } = setupAether({ empty: false });
      hit.mockResolvedValueOnce({ entry: 1, char: 4 });
      hit.mockResolvedValueOnce({ entry: 3, char: 2 });
      const { container } = renderLog();
      const div = getLogDiv(container);
      fireEvent.mouseDown(div, { button: 0 });
      await act(async () => {});
      fireEvent.mouseMove(window, { buttons: 1, clientX: 10, clientY: 90 });
      await waitFor(() =>
        expect(applied(setState)).toMatchObject({
          selectionStart: 1,
          selectionStartChar: 4,
          selectionEnd: 3,
          selectionEndChar: 2,
        }),
      );
    });

    it("should end a drag at a move whose hit resolves after the release", async () => {
      const { setState, hit } = setupAether({ empty: false });
      hit.mockResolvedValueOnce({ entry: 1, char: 4 });
      let resolveMove: (point: { entry: number; char: number }) => void = () => {};
      hit.mockReturnValueOnce(
        new Promise((resolve) => {
          resolveMove = resolve;
        }),
      );
      const { container } = renderLog();
      const div = getLogDiv(container);
      fireEvent.mouseDown(div, { button: 0 });
      await act(async () => {});
      fireEvent.mouseMove(window, { buttons: 1, clientX: 10, clientY: 90 });
      fireEvent.mouseUp(window);
      resolveMove({ entry: 3, char: 2 });
      await waitFor(() =>
        expect(applied(setState)).toMatchObject({
          selectionEnd: 3,
          selectionEndChar: 2,
        }),
      );
    });

    it("should stop selecting when the button comes up", async () => {
      const { hit } = setupAether({ empty: false });
      const { container } = renderLog();
      const div = getLogDiv(container);
      fireEvent.mouseDown(div, { button: 0 });
      await act(async () => {});
      fireEvent.mouseUp(window);
      fireEvent.mouseMove(window, { buttons: 1, clientY: 80 });
      expect(hit).toHaveBeenCalledTimes(1);
    });

    it("should not drag after a release that comes before the hit", async () => {
      const { hit } = setupAether({ empty: false });
      let resolve: (point: { entry: number; char: number }) => void = () => {};
      hit.mockReturnValueOnce(new Promise((r) => (resolve = r)));
      const { container } = renderLog();
      const div = getLogDiv(container);
      fireEvent.mouseDown(div, { button: 0 });
      fireEvent.mouseUp(window);
      resolve({ entry: 2, char: 5 });
      await act(async () => {});
      fireEvent.mouseMove(window, { buttons: 1, clientY: 80 });
      expect(hit).toHaveBeenCalledTimes(1);
    });

    it("should return to the pressed line when a drag comes back to its start", async () => {
      const { setState, hit } = setupAether({ empty: false });
      hit.mockResolvedValueOnce({ entry: 1, char: 4 });
      hit.mockResolvedValueOnce({ entry: 3, char: 2 });
      hit.mockResolvedValueOnce({ entry: 1, char: 4 });
      const { container } = renderLog();
      const div = getLogDiv(container);
      fireEvent.mouseDown(div, { button: 0 });
      await act(async () => {});
      fireEvent.mouseMove(window, { buttons: 1, clientY: 90 });
      await waitFor(() =>
        expect(applied(setState)).toMatchObject({ selectionEndChar: 2 }),
      );
      fireEvent.mouseMove(window, { buttons: 1, clientY: 30 });
      await waitFor(() =>
        expect(applied(setState)).toMatchObject({
          selectionStart: 1,
          selectionEnd: 1,
          ...WHOLE,
        }),
      );
    });

    it("should keep the start the worker moved as entries shift", async () => {
      const { setState, hit } = setupAether({ empty: false });
      hit.mockResolvedValueOnce({ entry: 5, char: 4 });
      hit.mockResolvedValueOnce({ entry: 8, char: 2 });
      const { container } = renderLog();
      const div = getLogDiv(container);
      fireEvent.mouseDown(div, { button: 0 });
      await act(async () => {});
      fireEvent.mouseMove(window, { buttons: 1, clientY: 90 });
      // Two entries were evicted, so the worker moved the start from 5 to 3.
      const shifted = { ...DEFAULT_STATE, selectionStart: 3, selectionStartChar: 4 };
      await waitFor(() =>
        expect(lastState(setState, shifted)).toMatchObject({
          selectionStart: 3,
          selectionStartChar: 4,
          selectionEnd: 8,
          selectionEndChar: 2,
        }),
      );
    });

    it("should keep the start character the worker moved", async () => {
      const { setState, hit } = setupAether({ empty: false });
      hit.mockResolvedValueOnce({ entry: 5, char: 4 });
      hit.mockResolvedValueOnce({ entry: 8, char: 2 });
      const { container } = renderLog();
      const div = getLogDiv(container);
      fireEvent.mouseDown(div, { button: 0 });
      await act(async () => {});
      fireEvent.mouseMove(window, { buttons: 1, clientY: 90 });
      // The start line was evicted, so the worker moved the start to the oldest line.
      const evicted = { ...DEFAULT_STATE, selectionStart: 0, selectionStartChar: 0 };
      await waitFor(() =>
        expect(lastState(setState, evicted)).toMatchObject({
          selectionStart: 0,
          selectionStartChar: 0,
          selectionEnd: 8,
        }),
      );
    });

    it("should end a drag whose selection the worker cleared", async () => {
      const { setState, hit } = setupAether({ empty: false });
      hit.mockResolvedValueOnce({ entry: 5, char: 4 });
      hit.mockResolvedValueOnce({ entry: 8, char: 2 });
      const { container } = renderLog();
      const div = getLogDiv(container);
      fireEvent.mouseDown(div, { button: 0 });
      await act(async () => {});
      // The press selected its line, then the worker evicted it and cleared the
      // selection.
      lastState(setState);
      fireEvent.mouseMove(window, { buttons: 1, clientY: 90 });
      await waitFor(() => expect(hit).toHaveBeenCalledTimes(2));
      await waitFor(() => expect(lastState(setState)).toEqual(DEFAULT_STATE));
      fireEvent.mouseMove(window, { buttons: 1, clientY: 80 });
      expect(hit).toHaveBeenCalledTimes(2);
    });

    it("should start no drag from a click that clears its line", async () => {
      const selected = {
        ...DEFAULT_STATE,
        ...WHOLE,
        selectionStart: 2,
        selectionEnd: 2,
      };
      const { setState, hit } = setupAether({ ...selected, empty: false });
      hit.mockResolvedValue({ entry: 2, char: 5 });
      const { container } = renderLog();
      const div = getLogDiv(container);
      fireEvent.mouseDown(div, { button: 0 });
      await act(async () => {});
      expect(lastState(setState, selected)).toMatchObject({ selectionStart: -1 });
      fireEvent.mouseMove(window, { buttons: 1, clientY: 90 });
      expect(hit).toHaveBeenCalledTimes(1);
    });

    it("should start no drag on a shift-click", async () => {
      const { hit } = setupAether({ empty: false });
      hit.mockResolvedValue({ entry: 4, char: 3 });
      const { container } = renderLog();
      const div = getLogDiv(container);
      fireEvent.mouseDown(div, { button: 0, shiftKey: true });
      await act(async () => {});
      fireEvent.mouseMove(window, { buttons: 1, clientY: 90 });
      expect(hit).toHaveBeenCalledTimes(1);
    });

    it("should extend the selection by whole lines on a shift-click", async () => {
      const { setState, hit } = setupAether({ empty: false });
      hit.mockResolvedValue({ entry: 4, char: 3 });
      const { container } = renderLog();
      fireEvent.mouseDown(getLogDiv(container), { button: 0, shiftKey: true });
      await waitFor(() =>
        expect(lastState(setState, { selectionStart: 1 })).toMatchObject({
          selectionStart: 1,
          selectionEnd: 4,
          ...WHOLE,
        }),
      );
    });
  });

  describe("drag past an edge", () => {
    const ROWS = {
      empty: false,
      computedLineHeight: 16,
      rowEntries: [10, 11, 12],
      region: box.construct({ x: 0, y: 0 }, { x: 400, y: 500 }),
    };

    beforeEach(() => {
      vi.useFakeTimers({ toFake: ["requestAnimationFrame", "cancelAnimationFrame"] });
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    const applyCalls = (
      setState: ReturnType<typeof vi.fn>,
      from: number,
      state: Record<string, unknown>,
    ): Record<string, unknown> =>
      setState.mock.calls
        .slice(from)
        .reduce(
          (s: Record<string, unknown>, [set]) =>
            typeof set === "function" ? set(s) : set,
          state,
        );

    const advanceFrames = (frames: number) => {
      for (let i = 0; i < frames; i++)
        act(() => {
          vi.advanceTimersToNextFrame();
        });
    };

    // Presses inside the log and waits for the worker to place the press.
    const press = async (container: HTMLElement): Promise<void> => {
      fireEvent.mouseDown(getLogDiv(container), { button: 0, clientY: 30 });
      await act(async () => {});
    };

    // Applies the state updates the log made from the drag on, in order.
    const dragTo = async (clientY: number, frames: number, props = {}) => {
      const { setState, state, hit } = setupAether(ROWS);
      hit.mockResolvedValueOnce({ entry: 10, char: 1 });
      hit.mockResolvedValueOnce({ entry: 12, char: 3 });
      const { container } = renderLog(props);
      const before = setState.mock.calls.length;
      await press(container);
      fireEvent.mouseMove(window, { clientY, buttons: 1 });
      await act(async () => {});
      advanceFrames(frames);
      return applyCalls(setState, before, state);
    };

    it("should scroll up and select from the start of the top row", async () => {
      const state = await dragTo(-50, 3);
      expect(state.wheelPos).toBeGreaterThan(0);
      expect(state.scrolling).toBe(true);
      expect(state.selectionEnd).toBe(aetherLog.selectRow(0));
      expect(state.selectionEndChar).toBe(0);
    });

    it("should pause the log when it scrolls up", async () => {
      const onHold = vi.fn();
      await dragTo(-50, 1, { onHold });
      expect(onHold).toHaveBeenCalledWith(true);
    });

    it("should scroll down and select to the end of the bottom row", async () => {
      const state = await dragTo(550, 3);
      expect(state.wheelPos).toBeLessThan(0);
      expect(state.selectionEnd).toBe(aetherLog.selectRow(34));
      expect(state.selectionEndChar).toBeGreaterThan(1_000);
    });

    it("should scroll faster the farther the pointer is past the edge", async () => {
      const near = (await dragTo(-10, 1)).wheelPos;
      const far = (await dragTo(-100, 1)).wheelPos;
      expect(far).toBeGreaterThan(Number(near));
    });

    it("should select to the character under a pointer inside the log", async () => {
      const state = await dragTo(100, 3);
      expect(state.wheelPos).toBe(0);
      expect(state).toMatchObject({ selectionEnd: 12, selectionEndChar: 3 });
    });

    it("should mark the log as selecting until mouse up", () => {
      const { setState, state } = setupAether(ROWS);
      const { container } = renderLog();
      fireEvent.mouseDown(getLogDiv(container), { button: 0, clientY: 30 });
      expect(applyCalls(setState, 0, state).selecting).toBe(true);
      fireEvent.mouseUp(window);
      expect(applyCalls(setState, 0, state).selecting).toBe(false);
    });

    it("should stop scrolling on mouse up", async () => {
      const { setState } = setupAether(ROWS);
      const { container } = renderLog();
      await press(container);
      fireEvent.mouseMove(window, { clientY: -50, buttons: 1 });
      fireEvent.mouseUp(window);
      const callsAfterUp = setState.mock.calls.length;
      act(() => {
        vi.advanceTimersToNextFrame();
      });
      expect(setState.mock.calls.length).toBe(callsAfterUp);
    });

    it("should stop scrolling when a move arrives with no button held", async () => {
      const { setState } = setupAether(ROWS);
      const { container } = renderLog();
      await press(container);
      fireEvent.mouseMove(window, { clientY: -50, buttons: 1 });
      fireEvent.mouseMove(window, { clientY: -60, buttons: 0 });
      const callsAfterRelease = setState.mock.calls.length;
      advanceFrames(1);
      fireEvent.mouseMove(window, { clientY: 80, buttons: 1 });
      expect(setState.mock.calls.length).toBe(callsAfterRelease);
    });

    it("should stop scrolling when the window loses focus", async () => {
      const { setState } = setupAether(ROWS);
      const { container } = renderLog();
      await press(container);
      fireEvent.mouseMove(window, { clientY: -50, buttons: 1 });
      fireEvent.blur(window);
      const callsAfterBlur = setState.mock.calls.length;
      advanceFrames(1);
      expect(setState.mock.calls.length).toBe(callsAfterBlur);
    });
  });

  describe("jump", () => {
    const Jumper = (): ReactElement => {
      const { jump, status } = Log.useJump();
      return <button onClick={() => jump("previous")}>{`Jump ${status}`}</button>;
    };

    it("should pause the log and request a jump in one update", () => {
      const { setState } = setupAether({ empty: false });
      const onHold = vi.fn();
      renderLog({ children: <Jumper />, hold: false, onHold, search: { text: "err" } });
      fireEvent.click(screen.getByRole("button", { name: "Jump" }));
      const update = setState.mock.calls.at(-1)?.[0];
      expect(update(DEFAULT_STATE)).toMatchObject({
        scrolling: true,
        jump: { direction: "previous", request: 1 },
      });
      expect(onHold).toHaveBeenCalledWith(true);
    });

    it("should neither pause nor jump for a search with no pattern", () => {
      const { setState } = setupAether({ empty: false });
      const onHold = vi.fn();
      renderLog({
        children: <Jumper />,
        hold: false,
        onHold,
        search: { text: "(", mode: "regex" },
      });
      setState.mockClear();
      fireEvent.click(screen.getByRole("button", { name: "Jump" }));
      expect(setState).not.toHaveBeenCalled();
      expect(onHold).not.toHaveBeenCalled();
    });

    it("should give children the status of the jump", () => {
      setupAether({ empty: false, jumpStatus: "none" });
      renderLog({ children: <Jumper /> });
      expect(screen.getByText("Jump none")).toBeDefined();
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
    const SELECTED = { empty: false, selectionStart: 0, rowEntries: [0, 1] };

    interface Item {
      data: Record<string, Promise<Blob>>;
    }

    const Messages = (): ReactElement => {
      const { statuses } = Status.useNotifications();
      return (
        <>
          {statuses.map(({ key, variant, message, description }) => (
            <span key={key}>
              <span>{`${variant}: ${message}`}</span>
              {description != null && <span>{description}</span>}
            </span>
          ))}
        </>
      );
    };

    const stubClipboard = () => {
      const write = vi.fn(async (_items: Item[]) => {});
      Object.defineProperty(globalThis, "ClipboardItem", {
        configurable: true,
        value: class implements Item {
          constructor(readonly data: Item["data"]) {}
        },
      });
      Object.defineProperty(navigator, "clipboard", {
        configurable: true,
        value: { write },
      });
      return write;
    };

    it("should leave the copy event alone when nothing is selected", () => {
      const { selection } = setupAether({ empty: false });
      const { container } = renderLog();
      expect(fireEvent.copy(getLogDiv(container))).toBe(true);
      expect(selection).not.toHaveBeenCalled();
    });

    it("should leave the copy event alone when the log has no rows", () => {
      setupAether({ ...SELECTED, selectionEnd: 0, rowEntries: [] });
      const { container } = renderLog();
      expect(fireEvent.copy(getLogDiv(container))).toBe(true);
    });

    it("should start the write within the copy event", async () => {
      const write = stubClipboard();
      setupAether({ ...SELECTED, selectionEnd: 1 });
      const { container } = renderLog();
      await act(async () => {
        expect(fireEvent.copy(getLogDiv(container))).toBe(false);
        expect(write).toHaveBeenCalledTimes(1);
      });
    });

    it("should read the lines from the worker only on a copy", async () => {
      const write = stubClipboard();
      const { selection } = setupAether({ ...SELECTED, selectionEnd: 1 });
      const { container } = renderLog();
      expect(selection).not.toHaveBeenCalled();
      await act(async () => {
        fireEvent.copy(getLogDiv(container));
      });
      expect(selection).toHaveBeenCalledTimes(1);
      const [item] = write.mock.calls[0][0];
      const text = await (await item.data["text/plain"]).text();
      expect(text).toBe("<one>\ntwo");
      const html = await (await item.data["text/html"]).text();
      expect(html).toContain("&lt;one&gt;");
      expect(html).toContain('<span style="color: #ff0000">two</span>');
    });

    it("should copy plain text in a context with no clipboard API", async () => {
      Reflect.deleteProperty(globalThis, "ClipboardItem");
      stubClipboardUnavailable();
      const copied = stubCopyCommand();
      setupAether({ ...SELECTED, selectionEnd: 1 });
      const { container } = renderLog({ children: <Messages /> });
      await act(async () => {
        expect(fireEvent.copy(getLogDiv(container))).toBe(false);
      });
      expect(copied).toHaveBeenCalledWith("<one>\ntwo");
      expect(
        await screen.findByText("success: Copied 2 lines to clipboard"),
      ).toBeDefined();
    });

    it("should report a copy the browser refuses", async () => {
      Reflect.deleteProperty(globalThis, "ClipboardItem");
      stubClipboardUnavailable();
      stubCopyCommand(false);
      setupAether({ ...SELECTED, selectionEnd: 1 });
      const { container } = renderLog({ children: <Messages /> });
      await act(async () => {
        fireEvent.copy(getLogDiv(container));
      });
      expect(
        await screen.findByText("error: Failed to copy to clipboard"),
      ).toBeDefined();
    });

    it("should report the count of copied lines", async () => {
      stubClipboard();
      setupAether({ ...SELECTED, selectionEnd: 1 });
      const { container } = renderLog({ children: <Messages /> });
      await act(async () => {
        fireEvent.copy(getLogDiv(container));
      });
      expect(
        await screen.findByText("success: Copied 2 lines to clipboard"),
      ).toBeDefined();
    });

    it("should warn that a select-all copy leaves out stored values", async () => {
      stubClipboard();
      setupAether({ ...SELECTED, selectionEnd: Number.MAX_SAFE_INTEGER }, true);
      const { container } = renderLog({ children: <Messages /> });
      await act(async () => {
        fireEvent.copy(getLogDiv(container));
      });
      expect(
        await screen.findByText("warning: Copied 2 lines to clipboard"),
      ).toBeDefined();
      expect(screen.getByText(/not every stored value/)).toBeDefined();
    });
  });

  describe("channel name resolution", () => {
    it("should pass channels to aether state", () => {
      const channels = [1, 2].map((channel) => log.channelEntryZ.parse({ channel }));
      renderLog({ channels });
      expect(mockAetherUse).toHaveBeenCalled();
      expect(getAetherInitialState().channels).toEqual(channels);
    });
  });

  describe("props forwarding", () => {
    it("should pass channelNamesHidden to aether state", () => {
      renderLog({ channelNamesHidden: true });
      expect(getAetherInitialState().channelNamesHidden).toBe(true);
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
      const channels = [
        log.channelEntryZ.parse({
          channel: 1,
          color: { type: "fixed", value: "#ff0000" },
        }),
        log.channelEntryZ.parse({ channel: 2 }),
      ];
      renderLog({ channels });
      expect(getAetherInitialState().channels).toEqual(channels);
    });
  });

  describe("keyboard triggers", () => {
    const PRIMED_STATE = {
      ...DEFAULT_STATE,
      empty: false,
      selectionStart: 2,
      selectionEnd: 7,
    };

    const findUpdaterResult = (
      setState: ReturnType<typeof vi.fn>,
      predicate: (result: Record<string, unknown>) => boolean,
    ): Record<string, unknown> | undefined => {
      for (const call of setState.mock.calls) {
        const updater = call[0];
        if (typeof updater !== "function") continue;
        const result = updater(PRIMED_STATE);
        if (predicate(result)) return result;
      }
      return undefined;
    };

    const fireCtrlA = () => {
      fireEvent.keyDown(document.body, { code: "ControlLeft" });
      fireEvent.keyDown(document.body, { code: "KeyA", ctrlKey: true });
      fireEvent.keyUp(document.body, { code: "KeyA", ctrlKey: true });
      fireEvent.keyUp(document.body, { code: "ControlLeft" });
    };

    const isSelectAll = (r: Record<string, unknown>) =>
      r.selectionStart === 0 &&
      r.selectionEnd === Number.MAX_SAFE_INTEGER &&
      r.selectionStartChar === -1 &&
      r.selectionEndChar === -1;
    const isClearSelection = (r: Record<string, unknown>) =>
      r.selectionStart === -1 && r.selectionEnd === -1;

    it("should fire setState selecting all entries on Ctrl+A when enableTriggers is true", () => {
      const { setState } = setupAether({ empty: false });
      renderLog({ enableTriggers: true });
      fireCtrlA();
      expect(findUpdaterResult(setState, isSelectAll)).toBeDefined();
    });

    it("should not fire setState on Ctrl+A when enableTriggers returns false", () => {
      const { setState } = setupAether({ empty: false });
      renderLog({ enableTriggers: () => false });
      fireCtrlA();
      expect(findUpdaterResult(setState, isSelectAll)).toBeUndefined();
    });

    it("should fire setState selecting all on Ctrl+A when enableTriggers is undefined", () => {
      const { setState } = setupAether({ empty: false });
      renderLog();
      fireCtrlA();
      expect(findUpdaterResult(setState, isSelectAll)).toBeDefined();
    });

    it("should clear selection on Escape when enableTriggers is true", () => {
      const { setState } = setupAether({ empty: false });
      renderLog({ enableTriggers: true });
      fireEvent.keyDown(document.body, { code: "Escape" });
      fireEvent.keyUp(document.body, { code: "Escape" });
      expect(findUpdaterResult(setState, isClearSelection)).toBeDefined();
    });

    it("should not clear selection on Escape when enableTriggers returns false", () => {
      const { setState } = setupAether({ empty: false });
      renderLog({ enableTriggers: () => false });
      fireEvent.keyDown(document.body, { code: "Escape" });
      fireEvent.keyUp(document.body, { code: "Escape" });
      expect(findUpdaterResult(setState, isClearSelection)).toBeUndefined();
    });
  });
});
