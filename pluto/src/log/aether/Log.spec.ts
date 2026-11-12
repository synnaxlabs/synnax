// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log as clientLog } from "@synnaxlabs/client";
import { theme } from "@synnaxlabs/lyra/theme";
import { box, color, TimeRange, TimeStamp } from "@synnaxlabs/x";
import { assert, beforeEach, describe, expect, it, vi } from "vitest";

import { Log, logStateZ, SELECT_ALL_END } from "@/log/aether/Log";
import {
  MockLogSource,
  mockLogSourceSpec,
  registerMockLogSource,
} from "@/log/aether/telem/mock";
import { type LogEntry } from "@/log/aether/telem/types";
import { telem } from "@/telem/aether";
import { renderAether } from "@/testutil/renderAether";
import { render } from "@/vis/render";
import { canvasTest } from "@/vis/render/testutil";

const THEME: theme.Theme = theme.themeZ.parse(theme.SYNNAX_DARK);

const REGION_500 = box.construct({ x: 0, y: 0 }, { width: 400, height: 500 });

const makeEntry = (i: number, channelKey: number = 1): LogEntry => ({
  channelKey,
  timestamp: TimeStamp.milliseconds(i * 1000),
  value: String(i),
});

const SCROLLBAR_WIDTH = 6;

// The thumb is the only rounded rect drawn at the scrollbar width; selection bands are
// unrounded and span the full region.
const findThumb = (recorder: ReturnType<typeof canvasTest.record>) => {
  const call = recorder.lower2d.calls.findLast(
    ({ op, args }) => op === "roundRect" && args[2] === SCROLLBAR_WIDTH,
  );
  if (call == null) return null;
  const [x, y, width, height] = call.args as number[];
  return { x, y, width, height };
};

const selectedText = (log: Log): string =>
  log
    .selection()
    .lines.map(({ text }) => text)
    .join("\n");

let testIdCounter = 0;

const baseInput = (
  testId: string,
  region: box.Box,
  overrides: Record<string, unknown>,
): Record<string, unknown> => ({
  region,
  wheelPos: 0,
  scrolling: false,
  empty: true,
  visible: true,
  telem: mockLogSourceSpec(testId),
  ...overrides,
});

// Mounts a Log under the real provider stack. The mock source is registered before
// mount so the telem TestFactory resolves it on the first afterUpdate; a second update
// runs after pushing entries so the Log reads them.
const setup = (
  entries: LogEntry[] = [],
  region: box.Box = REGION_500,
  stateOverrides: Record<string, unknown> = {},
  theme: theme.Theme = THEME,
) => {
  const testId = `log-test-${++testIdCounter}`;
  const source = new MockLogSource();
  registerMockLogSource(testId, source);
  const recorder = canvasTest.record();
  const input = baseInput(testId, region, stateOverrides);
  const h = renderAether(Log, {
    state: logStateZ.parse(input),
    theming: { theme, fontURLs: [] },
    render: recorder,
  });
  source.push(...entries);
  h.setState(() => logStateZ.parse(input));
  return {
    h,
    log: h.component,
    source,
    recorder,
    updateState: (overrides: Record<string, unknown>) =>
      h.setState(() => logStateZ.parse(baseInput(testId, region, overrides))),
  };
};

describe("log/aether/Log", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("lineHeight", () => {
    it("should calculate from theme typography and base size", () => {
      const { log } = setup();
      const expected = THEME.typography[log.state.font].size * THEME.sizes.base;
      expect(log.lineHeight).toBe(expected);
      expect(log.lineHeight).toBeGreaterThan(0);
    });
  });

  describe("visibleLineCount", () => {
    it("should calculate how many lines fit in the region", () => {
      const entries = Array.from({ length: 50 }, (_, i) => makeEntry(i));
      const { log } = setup(entries);
      const regionHeight = box.height(REGION_500);
      const expected = Math.min(
        Math.floor((regionHeight - 12) / log.lineHeight),
        entries.length,
      );
      expect(log.visibleLineCount).toBe(expected);
    });

    it("should be capped at entry count when fewer entries than viewport", () => {
      const entries = [makeEntry(0)];
      const { log } = setup(entries);
      expect(log.visibleLineCount).toBe(1);
    });
  });

  describe("totalHeight", () => {
    it("should be entries.length * lineHeight", () => {
      const entries = Array.from({ length: 10 }, (_, i) => makeEntry(i));
      const { log } = setup(entries);
      expect(log.totalHeight).toBeCloseTo(entries.length * log.lineHeight);
    });
  });

  describe("empty state detection", () => {
    it("should set empty to false when entries arrive", () => {
      const entries = [makeEntry(0)];
      const { log } = setup(entries);
      expect(log.state.empty).toBe(false);
    });

    it("should keep empty true when no entries", () => {
      const { log } = setup([]);
      expect(log.state.empty).toBe(true);
    });

    it("should set empty to false while the source loads", () => {
      const { log, source } = setup([]);
      source.status = telem.LOADING_STATUS;
      source.notify();
      expect(log.state.empty).toBe(false);
    });

    it("should be blank only while loading with no entries", () => {
      const { log, source } = setup([]);
      source.status = telem.LOADING_STATUS;
      source.notify();
      expect(log.state.blank).toBe(true);
      source.push(makeEntry(0));
      source.notify();
      expect(log.state.blank).toBe(false);
      expect(log.state.loading).toBe(true);
    });

    it("should set loading while the source loads", () => {
      const { log, source } = setup([]);
      expect(log.state.loading).toBe(false);
      source.status = telem.LOADING_STATUS;
      source.notify();
      expect(log.state.loading).toBe(true);
      source.status = telem.SUCCESS_STATUS;
      source.notify();
      expect(log.state.loading).toBe(false);
    });

    it("should set loading while it holds entries", () => {
      const { log, source } = setup([makeEntry(0)]);
      source.status = telem.LOADING_STATUS;
      source.notify();
      expect(log.state.loading).toBe(true);
      expect(log.state.empty).toBe(false);
    });

    it("should set empty to true when the source loads no entries", () => {
      const { log, source } = setup([]);
      source.status = telem.LOADING_STATUS;
      source.notify();
      source.status = telem.SUCCESS_STATUS;
      source.notify();
      expect(log.state.empty).toBe(true);
    });
  });

  describe("scrollback", () => {
    it("should initialize scrollState with zero values", () => {
      const { log } = setup();
      expect(log.scrollState.offset).toBe(0);
      expect(log.scrollState.tail).toBe(0);
      expect(log.scrollState.scrollRef).toBe(0);
    });

    it("should enter scrollback when scrolling transitions from false to true", () => {
      const entries = Array.from({ length: 100 }, (_, i) => makeEntry(i));
      const { log, updateState } = setup(entries);

      updateState({
        region: REGION_500,
        wheelPos: 0,
        scrolling: false,
        empty: true,
        visible: true,
      });

      updateState({
        region: REGION_500,
        wheelPos: 100,
        scrolling: true,
        empty: false,
        visible: true,
      });

      expect(log.scrollState.offset).toBe(entries.length);
      expect(log.scrollState.tail).toBe(0);
      expect(log.scrollState.scrollRef).toBe(100);
    });

    it("should not carry scrollback state into a later log", () => {
      const entries = Array.from({ length: 100 }, (_, i) => makeEntry(i));
      // A log mounted already paused runs the scrollback branch on its first update.
      setup(entries, REGION_500, { scrolling: true, empty: false });
      const { log } = setup(entries);
      expect(log.scrollState).toEqual({
        offset: 0,
        tail: 0,
        scrollRef: 0,
        moved: 0,
        awayFromEnd: false,
      });
    });

    it("should stay paused while new entries arrive after a pause at the newest entry", () => {
      const entries = Array.from({ length: 100 }, (_, i) => makeEntry(i));
      const { log, source, updateState } = setup(entries);
      updateState({ scrolling: false, empty: false });
      updateState({ scrolling: true, empty: false });
      source.push(...Array.from({ length: 20 }, (_, i) => makeEntry(100 + i)));
      updateState({ scrolling: true, empty: false });
      expect(log.state.scrolling).toBe(true);
      expect(log.scrollState.offset).toBe(100);
    });

    it("should stay paused when the pause begins at the newest entry", () => {
      const entries = Array.from({ length: 100 }, (_, i) => makeEntry(i));
      const { log, updateState } = setup(entries);
      updateState({ scrolling: false, empty: false });
      updateState({ scrolling: true, empty: false });
      updateState({ scrolling: true, empty: false });
      expect(log.state.scrolling).toBe(true);
    });

    it("should resume once the viewport returns to the newest entry", () => {
      const entries = Array.from({ length: 100 }, (_, i) => makeEntry(i));
      const { log, updateState } = setup(entries);
      updateState({ scrolling: false, empty: false });
      updateState({ scrolling: true, empty: false, wheelPos: 0 });
      updateState({ scrolling: true, empty: false, wheelPos: 300 });
      expect(log.state.scrolling).toBe(true);
      updateState({ scrolling: true, empty: false, wheelPos: 0 });
      expect(log.state.scrolling).toBe(false);
    });

    it("should resume the source when it resumes on its own", () => {
      const entries = Array.from({ length: 100 }, (_, i) => makeEntry(i));
      const { source, updateState } = setup(entries);
      updateState({ scrolling: false, empty: false });
      updateState({ scrolling: true, empty: false, wheelPos: 0 });
      updateState({ scrolling: true, empty: false, wheelPos: 300 });
      expect(source.resumeCalls).toBe(0);
      updateState({ scrolling: true, empty: false, wheelPos: 0 });
      expect(source.resumeCalls).toBeGreaterThan(0);
    });

    it("should clear the loading state of a read that resuming cancels", () => {
      const entries = Array.from({ length: 100 }, (_, i) => makeEntry(i));
      const { log, source, updateState } = setup(entries);
      updateState({ scrolling: false, empty: false });
      updateState({ scrolling: true, empty: false, wheelPos: 0 });
      updateState({ scrolling: true, empty: false, wheelPos: 300 });
      const { status } = source;
      source.status = { ...status, variant: "loading" };
      source.resume = () => {
        source.status = status;
      };
      updateState({ scrolling: true, empty: false, wheelPos: 0 });
      expect(log.state.loading).toBe(false);
    });

    it("should stamp resumedAt only when it resumes on its own", () => {
      const entries = Array.from({ length: 100 }, (_, i) => makeEntry(i));
      const { log, updateState } = setup(entries);
      updateState({ scrolling: false, empty: false });
      updateState({ scrolling: true, empty: false, wheelPos: 0 });
      updateState({ scrolling: true, empty: false, wheelPos: 300 });
      expect(log.state.resumedAt).toBe(0);
      updateState({ scrolling: true, empty: false, wheelPos: 0 });
      expect(log.state.resumedAt).toBeGreaterThan(0);
    });
  });

  describe("entries", () => {
    it("should store entries from telem source", () => {
      const entries = Array.from({ length: 5 }, (_, i) => makeEntry(i));
      const { log } = setup(entries);
      expect(log.entries).toHaveLength(5);
      expect(log.entries[0].value).toBe("0");
      expect(log.entries[4].value).toBe("4");
    });
  });

  describe("logStateZ schema", () => {
    it("should provide defaults for new selection fields", () => {
      const parsed = logStateZ.parse({
        region: REGION_500,
        wheelPos: 0,
        scrolling: false,
        empty: true,
        visible: true,
      });
      expect(parsed.selectionStart).toBe(-1);
      expect(parsed.selectionEnd).toBe(-1);
      expect(parsed.computedLineHeight).toBe(0);
    });

    it("should provide defaults for channel-related fields", () => {
      const parsed = logStateZ.parse({
        region: REGION_500,
        wheelPos: 0,
        scrolling: false,
        empty: true,
        visible: true,
      });
      expect(parsed.channelNamesHidden).toBe(false);
      expect(parsed.timestampPrecision).toBe(0);
      expect(parsed.channelNames).toEqual({});
      expect(parsed.channels).toEqual([]);
    });

    it("should accept explicit channel config values", () => {
      const parsed = logStateZ.parse({
        region: REGION_500,
        wheelPos: 0,
        scrolling: false,
        empty: true,
        visible: true,
        channels: [
          clientLog.channelEntryZ.parse({
            channel: 1,
            color: color.construct("#ff0000"),
            precision: 3,
          }),
          clientLog.channelEntryZ.parse({ channel: 2, color: color.ZERO }),
        ],
        channelNamesHidden: true,
        timestampPrecision: 2,
      });
      expect(parsed.channelNamesHidden).toBe(true);
      expect(parsed.timestampPrecision).toBe(2);
      expect(parsed.channels).toHaveLength(2);
      expect(parsed.channels[0].color).toEqual([255, 0, 0, 1]);
      expect(parsed.channels[0].precision).toBe(3);
    });
  });

  describe("timestampPrecision", () => {
    it("should default to 0 and produce 8-char timestamps", () => {
      const { log } = setup();
      expect(log.state.timestampPrecision).toBe(0);
    });

    it("should accept precision values 0-3", () => {
      const { log } = setup([], REGION_500, { timestampPrecision: 3 });
      expect(log.state.timestampPrecision).toBe(3);
    });
  });

  describe("channel management", () => {
    it("should pass channel keys to telem source", () => {
      const { source } = setup([], REGION_500, {
        channels: [
          clientLog.channelEntryZ.parse({ channel: 1, color: color.ZERO }),
          clientLog.channelEntryZ.parse({ channel: 2, color: color.ZERO }),
          clientLog.channelEntryZ.parse({ channel: 3, color: color.ZERO }),
        ],
      });
      // afterUpdate calls setChannels on the source. Verify the filter works
      // by pushing entries for configured and unconfigured channels.
      source.push(makeEntry(0, 1), makeEntry(1, 2), makeEntry(2, 99));
      const keys = source.value().data.map((e) => e.channelKey);
      expect(keys).toContain(1);
      expect(keys).toContain(2);
      expect(keys).not.toContain(99);
    });

    it("should not show the entries of a hidden channel", () => {
      const entries = Array.from({ length: 10 }, (_, i) => makeEntry(i, (i % 2) + 1));
      const { log } = setup(entries, REGION_500, { hiddenChannels: [2] });
      expect(log.entries.map((e) => e.channelKey)).toEqual(Array(5).fill(1));
    });

    it("should return a paused view to its time when a channel hides", () => {
      const entries = Array.from({ length: 400 }, (_, i) => makeEntry(i, (i % 2) + 1));
      const { log, source, updateState } = setup(entries);
      const base = { region: REGION_500, empty: false, visible: true };
      updateState({ ...base, wheelPos: 0, scrolling: false });
      updateState({ ...base, wheelPos: 0, scrolling: true });
      updateState({ ...base, wheelPos: 2000, scrolling: true });
      const time = log.entries[log.scrollState.offset - 1].timestamp.valueOf();
      // The source reloads only its newest entries when the filter changes.
      source.setEntries(entries.slice(300));
      updateState({ ...base, wheelPos: 2000, scrolling: true, hiddenChannels: [2] });
      expect(source.loadOlderCalls).toBeGreaterThan(0);
      source.setEntries(entries);
      source.notify();
      const shown = log.entries.filter((e) => e.timestamp.valueOf() <= time);
      expect(log.scrollState.offset).toBe(shown.length);
    });

    it("should stay paused when every channel hides", () => {
      const entries = Array.from({ length: 400 }, (_, i) => makeEntry(i, (i % 2) + 1));
      const { log, source, updateState } = setup(entries);
      const base = { region: REGION_500, empty: false, visible: true };
      updateState({ ...base, wheelPos: 0, scrolling: false });
      updateState({ ...base, wheelPos: 0, scrolling: true });
      updateState({ ...base, wheelPos: 2000, scrolling: true });
      const hidden = {
        ...base,
        wheelPos: 2000,
        scrolling: true,
        hiddenChannels: [1, 2],
      };
      updateState(hidden);
      updateState(hidden);
      expect(log.state.scrolling).toBe(true);
      expect(source.resumeCalls).toBe(0);
    });

    it("should stay paused when the shown entries fit on one screen", () => {
      const entries = Array.from({ length: 400 }, (_, i) => makeEntry(i, (i % 2) + 1));
      const { log, source, updateState } = setup(entries);
      const base = { region: REGION_500, empty: false, visible: true };
      updateState({ ...base, wheelPos: 0, scrolling: false });
      updateState({ ...base, wheelPos: 0, scrolling: true });
      updateState({ ...base, wheelPos: 2000, scrolling: true });
      source.setEntries(entries.slice(394));
      source.exhausted = true;
      const hidden = { ...base, wheelPos: 2000, scrolling: true, hiddenChannels: [2] };
      updateState(hidden);
      updateState(hidden);
      expect(log.state.scrolling).toBe(true);
      expect(source.resumeCalls).toBe(0);
    });
  });

  describe("search", () => {
    it("should filter the source by a search with its filter on", () => {
      const { source, updateState } = setup();
      updateState({ search: { text: "err", filtered: true } });
      expect(source.filter?.search).toEqual({ text: "err", mode: "text" });
      updateState({ search: { text: "err" } });
      expect(source.filter?.search).toBeNull();
    });

    it("should hand the source the format each channel draws with", () => {
      const { source, updateState } = setup();
      const channels = [
        clientLog.channelEntryZ.parse({ channel: 1, color: color.ZERO, precision: 2 }),
      ];
      const channelDataTypes = { "1": "float64" };
      updateState({
        channels,
        channelDataTypes,
        search: { text: "1.50", filtered: true },
      });
      expect(source.filter.formats["1"]).toMatchObject({
        precision: 2,
        dataType: "float64",
      });
    });

    it("should filter nothing with an invalid regex", () => {
      const { source, updateState } = setup();
      updateState({ search: { text: "(", mode: "regex", filtered: true } });
      expect(source.filter?.search).toBeNull();
    });

    it("should report how far back a filtered log has searched", () => {
      const { log, source, updateState } = setup([makeEntry(5)]);
      source.reached = TimeStamp.seconds(3);
      updateState({ search: { text: "5", filtered: true } });
      expect(log.state.searchedTo).toBe(TimeStamp.seconds(3).valueOf().toString());
      source.exhausted = true;
      source.notify();
      expect(log.state.searchedTo).toBe("");
    });
  });

  describe("search highlights", () => {
    // The recorder measures each character 8 pixels wide.
    const CHAR = 8;
    const PADDING = 6;
    const BARE = { timestampsHidden: true, channelNamesHidden: true };

    // Returns the highlights drawn, in columns from the left edge of the text.
    const highlights = (
      value: string,
      search: Record<string, unknown>,
      overrides: Record<string, unknown> = BARE,
      region: box.Box = REGION_500,
    ): Array<{ column: number; columns: number }> => {
      const { log, recorder } = setup([{ ...makeEntry(0), value }], region, {
        ...overrides,
        search,
      });
      recorder.clear();
      log.render();
      return recorder.lower2d.calls
        .filter(({ op }) => op === "rect")
        .map(({ args }) => {
          const [x, , width] = args as number[];
          return { column: (x - PADDING) / CHAR, columns: width / CHAR };
        });
    };

    it("should highlight every match of plain text, in any case", () => {
      const drawn = highlights("an Error, then an error", { text: "error" });
      expect(drawn.map(({ column, columns }) => [column, columns])).toEqual([
        [3, 5],
        [18, 5],
      ]);
    });

    it("should highlight regex matches", () => {
      const drawn = highlights("a1 b22 c333", { text: "\\d+", mode: "regex" });
      expect(drawn.map(({ column, columns }) => [column, columns])).toEqual([
        [1, 1],
        [4, 2],
        [8, 3],
      ]);
    });

    it("should highlight with the filter off or on", () => {
      const off = highlights("abc", { text: "b" });
      const on = highlights("abc", { text: "b", filtered: true });
      expect(on).toEqual(off);
      expect(off).toHaveLength(1);
    });

    it("should highlight the number as the log draws it at its precision", () => {
      const drawn = highlights(
        "3.14159",
        { text: "3.14$", mode: "regex" },
        {
          ...BARE,
          channels: [
            clientLog.channelEntryZ.parse({
              channel: 1,
              color: color.ZERO,
              precision: 2,
            }),
          ],
          channelDataTypes: { "1": "float64" },
        },
      );
      expect(drawn.map(({ column, columns }) => [column, columns])).toEqual([[0, 4]]);
    });

    it("should draw nothing for an empty match", () => {
      expect(highlights("abc", { text: "^", mode: "regex" })).toEqual([]);
    });

    it("should highlight nothing for an invalid regex", () => {
      expect(highlights("a(b", { text: "(", mode: "regex" })).toEqual([]);
    });

    it("should start a highlight after the prefix", () => {
      const [drawn] = highlights(
        "abc",
        { text: "b" },
        { timestampsHidden: true, channelNamesHidden: false },
      );
      // The prefix is the channel's name and three spaces.
      expect(drawn.column).toBe("1   ".length + 1);
    });

    it("should give a wide character two columns", () => {
      const [drawn] = highlights("日本 err", { text: "err" });
      expect(drawn).toMatchObject({ column: 5, columns: 3 });
    });

    it("should place a match after the sign of a negative number", () => {
      const [drawn] = highlights("-123", { text: "2" });
      expect(drawn).toMatchObject({ column: 1, columns: 1 });
    });

    it("should split a match across the rows of a wrapped value", () => {
      // 10 columns of text fit in the region.
      const region = box.construct(
        { x: 0, y: 0 },
        { width: PADDING * 2 + 6 + 10 * CHAR, height: 500 },
      );
      const drawn = highlights("aaaaaaaabbbbcc", { text: "bbbb" }, BARE, region);
      expect(drawn.map(({ column, columns }) => [column, columns])).toEqual([
        [8, 2],
        [0, 2],
      ]);
    });
  });

  describe("color handling", () => {
    it("should use gray.l11 as text color when no custom color is set", () => {
      const { log } = setup([makeEntry(0)]);
      expect(color.isZero(log.state.color)).toBe(true);
    });

    it("should accept a custom text color", () => {
      const { log } = setup([], REGION_500, { color: "#ff0000" });
      expect(color.isZero(log.state.color)).toBe(false);
    });
  });

  describe("visibility", () => {
    it("should request render when visible", () => {
      const { recorder } = setup([makeEntry(0)], REGION_500, {
        visible: true,
      });
      expect(recorder.loopCalls.length).toBeGreaterThan(0);
    });

    it("should skip render when not visible and prevState also not visible", () => {
      const { recorder } = setup([makeEntry(0)], REGION_500, {
        visible: false,
      });
      // On the very first update, both state and prevState have visible=false,
      // so the early return on line 217 fires and no render is requested.
      expect(recorder.loopCalls).toHaveLength(0);
    });

    // Regression: the telem onChange handler requested a render unconditionally, so a
    // hidden log with live data enqueued a render (and its erase cleanup) per frame.
    it("should not request render when telem data arrives while hidden", () => {
      const { source, recorder } = setup([makeEntry(0)], REGION_500, {
        visible: false,
      });
      const before = recorder.loopCalls.length;
      source.push(makeEntry(1));
      source.notify();
      expect(recorder.loopCalls).toHaveLength(before);
    });

    it("should request render when telem data arrives while visible", () => {
      const { source, recorder } = setup([makeEntry(0)], REGION_500, {
        visible: true,
      });
      const before = recorder.loopCalls.length;
      source.push(makeEntry(1));
      source.notify();
      expect(recorder.loopCalls.length).toBeGreaterThan(before);
    });

    // A paused log whose entries all fit clamps its wheel position on every update.
    it("should request a render to erase a paused log when it hides", () => {
      const { h, recorder } = setup([makeEntry(0)]);
      h.setState((s) => ({ ...s, scrolling: true }));
      recorder.clear();
      h.setState((s) => ({ ...s, visible: false }));
      expect(recorder.loopCalls).toHaveLength(1);
    });
  });

  describe("scrollback with continued scrolling", () => {
    it("should adjust offset based on wheel position delta", () => {
      const entries = Array.from({ length: 100 }, (_, i) => makeEntry(i));
      const { log, updateState } = setup(entries);

      updateState({
        region: REGION_500,
        wheelPos: 0,
        scrolling: false,
        empty: true,
        visible: true,
      });

      updateState({
        region: REGION_500,
        wheelPos: 100,
        scrolling: true,
        empty: false,
        visible: true,
      });

      const initialOffset = log.scrollState.offset;
      expect(initialOffset).toBe(entries.length);

      updateState({
        region: REGION_500,
        wheelPos: 200,
        scrolling: true,
        empty: false,
        visible: true,
      });

      expect(log.scrollState.offset).toBeLessThanOrEqual(entries.length);
      expect(log.scrollState.offset).toBeGreaterThan(0);
    });

    it("should exit scrollback when offset reaches entry count", () => {
      const entries = Array.from({ length: 100 }, (_, i) => makeEntry(i));
      const { log, updateState } = setup(entries);

      updateState({
        region: REGION_500,
        wheelPos: 0,
        scrolling: false,
        empty: true,
        visible: true,
      });

      updateState({
        region: REGION_500,
        wheelPos: 100,
        scrolling: true,
        empty: false,
        visible: true,
      });

      updateState({
        region: REGION_500,
        wheelPos: -5000,
        scrolling: true,
        empty: false,
        visible: true,
      });

      expect(log.state.scrolling).toBe(false);
    });
  });

  describe("selection anchoring", () => {
    it("should adjust selection indices when entries are evicted", () => {
      const entries = Array.from({ length: 20 }, (_, i) => makeEntry(i));
      const { log, source, updateState } = setup(entries);

      updateState({
        region: REGION_500,
        wheelPos: 0,
        scrolling: false,
        empty: true,
        visible: true,
        selectionStart: 5,
        selectionEnd: 10,
      });

      source.setEntries(entries.slice(3));
      source.notify();

      expect(log.state.selectionStart).toBe(2);
      expect(log.state.selectionEnd).toBe(7);
    });

    it("should clear selection when all selected entries are evicted", () => {
      const entries = Array.from({ length: 20 }, (_, i) => makeEntry(i));
      const { log, source, updateState } = setup(entries);

      updateState({
        region: REGION_500,
        wheelPos: 0,
        scrolling: false,
        empty: true,
        visible: true,
        selectionStart: 0,
        selectionEnd: 2,
      });

      source.setEntries(entries.slice(5));
      source.notify();

      expect(log.state.selectionStart).toBe(-1);
      expect(log.state.selectionEnd).toBe(-1);
      expect(log.selection().lines).toEqual([]);
    });

    it("should not modify selection when no entries are evicted", () => {
      const entries = Array.from({ length: 20 }, (_, i) => makeEntry(i));
      const { log, source, updateState } = setup(entries);

      updateState({
        region: REGION_500,
        wheelPos: 0,
        scrolling: false,
        empty: true,
        visible: true,
        selectionStart: 5,
        selectionEnd: 10,
      });

      source.notify();

      expect(log.state.selectionStart).toBe(5);
      expect(log.state.selectionEnd).toBe(10);
    });

    it("should clamp selectionStart to 0 when partially evicted", () => {
      const entries = Array.from({ length: 20 }, (_, i) => makeEntry(i));
      const { log, source, updateState } = setup(entries);

      updateState({
        region: REGION_500,
        wheelPos: 0,
        scrolling: false,
        empty: true,
        visible: true,
        selectionStart: 2,
        selectionEnd: 8,
      });

      // Evict 4 entries — start goes negative, end stays positive
      source.setEntries(entries.slice(4));
      source.notify();

      expect(log.state.selectionStart).toBe(0);
      expect(log.state.selectionEnd).toBe(4);
    });

    it("should anchor a selection end past the last entry to the last entry", () => {
      const entries = Array.from({ length: 20 }, (_, i) => makeEntry(i));
      const { log, source, updateState } = setup(entries);
      updateState({
        region: REGION_500,
        wheelPos: 0,
        scrolling: false,
        empty: true,
        visible: true,
        selectionStart: 5,
        selectionEnd: 30,
      });
      source.setEntries(entries.slice(3));
      source.notify();
      expect(log.state.selectionStart).toBe(2);
      expect(log.state.selectionEnd).toBe(16);
    });

    it("should keep select-all open while entries are evicted", () => {
      const entries = Array.from({ length: 20 }, (_, i) => makeEntry(i));
      const { log, source, updateState } = setup(entries);
      updateState({
        region: REGION_500,
        wheelPos: 0,
        scrolling: false,
        empty: true,
        visible: true,
        selectionStart: 0,
        selectionEnd: SELECT_ALL_END,
      });
      source.setEntries([...entries.slice(3), makeEntry(20)]);
      source.notify();
      expect(log.state.selectionStart).toBe(0);
      expect(log.state.selectionEnd).toBe(SELECT_ALL_END);
    });

    it("should keep select-all on every entry when older entries land", () => {
      const entries = Array.from({ length: 20 }, (_, i) => makeEntry(i + 5));
      const { log, source, updateState } = setup(entries);
      updateState({ selectionStart: 0, selectionEnd: SELECT_ALL_END });
      const older = Array.from({ length: 5 }, (_, i) => makeEntry(i));
      source.setEntries([...older, ...entries]);
      source.notify();
      expect(log.state.selectionStart).toBe(0);
      expect(log.selection().lines).toHaveLength(25);
    });

    it("should follow the selected entries when an entry lands before them", () => {
      const entries = Array.from({ length: 20 }, (_, i) => makeEntry(i));
      const { log, source, updateState } = setup(entries);
      updateState({
        region: REGION_500,
        wheelPos: 0,
        scrolling: false,
        empty: true,
        visible: true,
        selectionStart: 5,
        selectionEnd: 10,
      });
      const earlier = { ...makeEntry(2), value: "earlier" };
      source.setEntries([...entries.slice(0, 3), earlier, ...entries.slice(3)]);
      source.notify();
      expect(log.state.selectionStart).toBe(6);
      expect(log.state.selectionEnd).toBe(11);
    });
  });

  describe("scrollback anchoring", () => {
    it("should reduce scroll offset when entries are evicted during scrollback", () => {
      const entries = Array.from({ length: 100 }, (_, i) => makeEntry(i));
      const { log, source, updateState } = setup(entries);

      updateState({
        region: REGION_500,
        wheelPos: 0,
        scrolling: false,
        empty: true,
        visible: true,
      });

      updateState({
        region: REGION_500,
        wheelPos: 100,
        scrolling: true,
        empty: false,
        visible: true,
      });

      const offsetBeforeEviction = log.scrollState.offset;

      source.setEntries(entries.slice(10));
      source.notify();

      expect(log.scrollState.offset).toBe(offsetBeforeEviction - 10);
    });

    it("should keep the paused view on its entries when an entry lands before them", () => {
      const entries = Array.from({ length: 100 }, (_, i) => makeEntry(i));
      const { log, source, updateState } = setup(entries);
      updateState({
        region: REGION_500,
        wheelPos: 0,
        scrolling: false,
        empty: true,
        visible: true,
      });
      updateState({
        region: REGION_500,
        wheelPos: 100,
        scrolling: true,
        empty: false,
        visible: true,
      });
      const before = log.scrollState.offset;
      const earlier = { ...makeEntry(10), value: "earlier" };
      source.setEntries([...entries.slice(0, 11), earlier, ...entries.slice(11)]);
      source.notify();
      expect(log.scrollState.offset).toBe(before + 1);
    });
  });

  describe("older entries", () => {
    const pause = (wheelPos: number) => {
      const entries = Array.from({ length: 1000 }, (_, i) => makeEntry(i));
      const { source, updateState } = setup(entries);
      const base = { region: REGION_500, empty: false, visible: true };
      updateState({ ...base, wheelPos: 0, scrolling: false });
      updateState({ ...base, wheelPos: 0, scrolling: true });
      updateState({ ...base, wheelPos, scrolling: true });
      return source;
    };

    it("should not read older entries while the view is far from the oldest", () => {
      expect(pause(0).loadOlderCalls).toBe(0);
    });

    it("should read older entries when the paused view nears the oldest", () => {
      expect(pause(20_000).loadOlderCalls).toBeGreaterThan(0);
    });

    it("should read newer entries when the paused view nears the newest", () => {
      expect(pause(0).loadNewerCalls).toBeGreaterThan(0);
    });

    it("should not read newer entries after the view moves far from the newest", () => {
      expect(pause(20_000).loadNewerCalls).toBe(1);
    });

    it("should stay paused at the newest entry of a source that is not live", () => {
      const entries = Array.from({ length: 1000 }, (_, i) => makeEntry(i));
      const { log, source, updateState } = setup(entries);
      source.live = false;
      const base = { region: REGION_500, empty: false, visible: true };
      updateState({ ...base, wheelPos: 0, scrolling: false });
      updateState({ ...base, wheelPos: 0, scrolling: true });
      updateState({ ...base, wheelPos: 200, scrolling: true });
      updateState({ ...base, wheelPos: 0, scrolling: true });
      expect(log.state.scrolling).toBe(true);
    });

    it("should resume the source when the log leaves its paused view", () => {
      const entries = Array.from({ length: 1000 }, (_, i) => makeEntry(i));
      const { source, updateState } = setup(entries);
      const base = { region: REGION_500, empty: false, visible: true };
      updateState({ ...base, wheelPos: 0, scrolling: false });
      updateState({ ...base, wheelPos: 0, scrolling: true });
      expect(source.resumeCalls).toBe(0);
      updateState({ ...base, wheelPos: 0, scrolling: false });
      expect(source.resumeCalls).toBe(1);
    });

    it("should not read older entries while live", () => {
      const entries = Array.from({ length: 10 }, (_, i) => makeEntry(i));
      const { source } = setup(entries);
      source.notify();
      expect(source.loadOlderCalls).toBe(0);
    });
  });

  describe("position", () => {
    const entries = Array.from({ length: 200 }, (_, i) => makeEntry(i));
    const positionOf = ({ timestamp }: LogEntry): string =>
      timestamp.valueOf().toString();

    it("should return a log that mounts paused to its position", () => {
      const { log, source } = setup(entries, REGION_500, {
        scrolling: true,
        position: positionOf(entries[120]),
      });
      source.notify();
      expect(log.scrollState).toMatchObject({ offset: 121, tail: 0 });
    });

    it("should stay at the last entry of a time that entries share", () => {
      const shared = [...entries.slice(0, 121), makeEntry(120), ...entries.slice(121)];
      const { log, source } = setup(shared, REGION_500, {
        scrolling: true,
        position: positionOf(entries[120]),
      });
      source.notify();
      expect(log.scrollState.offset).toBe(122);
    });

    it("should load older entries until they reach the position", () => {
      const { log, source } = setup(entries.slice(50), REGION_500, {
        scrolling: true,
        position: positionOf(entries[100]),
      });
      source.setEntries(entries.slice(150));
      source.notify();
      expect(source.loadOlderCalls).toBeGreaterThan(0);
      source.setEntries([...entries]);
      source.notify();
      expect(log.scrollState.offset).toBe(101);
    });

    it("should show the oldest entries when no stored sample reaches back", () => {
      const { log, source } = setup(entries.slice(150), REGION_500, {
        scrolling: true,
        position: positionOf(entries[100]),
      });
      source.exhausted = true;
      source.notify();
      log.render();
      const bottom = log.state.rowEntries.at(-1) ?? -1;
      expect(log.state.position).toBe(positionOf(entries[150 + bottom]));
    });

    it("should leave the view to a wheel that moves while the log waits", () => {
      const { h, log, source } = setup(entries.slice(150), REGION_500, {
        scrolling: true,
        position: positionOf(entries[100]),
      });
      source.notify();
      h.setState((s) => ({ ...s, wheelPos: s.wheelPos - log.lineHeight }));
      source.setEntries([...entries]);
      source.notify();
      expect(log.scrollState.offset).not.toBe(101);
    });

    it("should ignore the position of a log that is not paused", () => {
      const { log, source } = setup(entries, REGION_500, {
        position: positionOf(entries[120]),
      });
      source.notify();
      log.render();
      expect(log.state.rowEntries.at(-1)).toBe(199);
      expect(log.state.position).toBe("");
    });

    it("should report the entry at the bottom of a paused view", () => {
      const { h, log } = setup(entries);
      h.setState((s) => ({ ...s, scrolling: true }));
      h.setState((s) => ({ ...s, wheelPos: s.wheelPos + log.lineHeight * 3 }));
      log.render();
      expect(log.state.position).toBe(positionOf(entries[196]));
    });

    it("should clear the position when the log resumes", () => {
      const { h, log } = setup(entries);
      h.setState((s) => ({ ...s, scrolling: true }));
      log.render();
      h.setState((s) => ({ ...s, scrolling: false }));
      log.render();
      expect(log.state.position).toBe("");
    });
  });

  describe("after the 2D canvases are restored", () => {
    it("should request a render until deleted", () => {
      const { h, recorder } = setup([makeEntry(1)]);
      recorder.clear();
      recorder.restore2D();
      expect(recorder.loopCalls).toHaveLength(1);
      h.unmount();
      recorder.clear();
      recorder.restore2D();
      expect(recorder.loopCalls).toEqual([]);
    });

    it("should not request a render while hidden", () => {
      const { recorder, updateState } = setup([makeEntry(1)]);
      updateState({ visible: false });
      recorder.clear();
      recorder.restore2D();
      expect(recorder.loopCalls).toEqual([]);
    });
  });

  describe("render", () => {
    it("should return undefined for zero-area region", () => {
      const zeroRegion = box.construct({ x: 0, y: 0 }, { width: 0, height: 0 });
      const { log } = setup([makeEntry(0)], zeroRegion);
      const result = log.render();
      expect(result).toBeUndefined();
    });

    it("should return a cleanup function when not visible", () => {
      const { log } = setup([makeEntry(0)], REGION_500, {
        visible: false,
      });
      const result = log.render();
      expect(result).toBeTypeOf("function");
    });

    it("should return a cleanup function when visible with entries", () => {
      const entries = Array.from({ length: 10 }, (_, i) => makeEntry(i));
      const { log } = setup(entries);
      const result = log.render();
      expect(result).toBeTypeOf("function");
    });

    it("should update visibleStart state during render", () => {
      const entries = Array.from({ length: 50 }, (_, i) => makeEntry(i));
      const { log } = setup(entries);
      log.render();
      const expectedStart = entries.length - log.visibleLineCount + 1;
      expect(log.state.visibleStart).toBe(expectedStart);
    });

    it("should update computedLineHeight state during render", () => {
      const entries = Array.from({ length: 10 }, (_, i) => makeEntry(i));
      const { log } = setup(entries);
      log.render();
      expect(log.state.computedLineHeight).toBe(log.lineHeight);
    });
  });

  describe("render with scrollback", () => {
    it("should render a paused log whose new source has fewer entries", () => {
      const entries = Array.from({ length: 100 }, (_, i) => makeEntry(i));
      const { log, h, updateState } = setup(entries);
      updateState({ empty: false, wheelPos: 0 });
      updateState({ empty: false, wheelPos: 0, scrolling: true });
      const testId = `log-test-${++testIdCounter}`;
      const source = new MockLogSource();
      source.push(makeEntry(0), makeEntry(1));
      registerMockLogSource(testId, source);
      h.setState(() =>
        logStateZ.parse(
          baseInput(testId, REGION_500, { empty: false, wheelPos: 0, scrolling: true }),
        ),
      );
      expect(log.render()).toBeTypeOf("function");
      expect(log.state.rowEntries).toEqual([0, 0, 1]);
    });

    it("should render the correct slice when scrolled back", () => {
      const entries = Array.from({ length: 100 }, (_, i) => makeEntry(i));
      const { log, updateState } = setup(entries);

      updateState({
        region: REGION_500,
        wheelPos: 0,
        scrolling: false,
        empty: true,
        visible: true,
      });

      updateState({
        region: REGION_500,
        wheelPos: 100,
        scrolling: true,
        empty: false,
        visible: true,
      });

      const result = log.render();
      expect(result).toBeTypeOf("function");
      expect(log.state.visibleStart).toBeLessThan(entries.length);
    });
  });

  describe("channel configs and formatting", () => {
    it("should format entries with channel names when channelNamesHidden is false", () => {
      const entries = Array.from({ length: 5 }, (_, i) => makeEntry(i, 1));
      const { log } = setup(entries, REGION_500, {
        channelNamesHidden: false,
        channels: [clientLog.channelEntryZ.parse({ channel: 1, color: color.ZERO })],
        channelNames: { "1": "Sensor1" },
        selectionStart: 0,
        selectionEnd: 0,
      });
      log.render();
      expect(selectedText(log)).toContain("Sensor1");
    });

    it("should format entries without channel names when channelNamesHidden is true", () => {
      const entries = Array.from({ length: 5 }, (_, i) => makeEntry(i, 1));
      const { log } = setup(entries, REGION_500, {
        channelNamesHidden: true,
        selectionStart: 0,
        selectionEnd: 0,
      });
      log.render();
      expect(selectedText(log)).not.toContain("Sensor1");
    });

    it("should put two spaces after the time and three before the value", () => {
      const entries: LogEntry[] = [
        { channelKey: 1, timestamp: TimeStamp.milliseconds(1000), value: "v" },
      ];
      const { log } = setup(entries, REGION_500, {
        channelNamesHidden: false,
        channels: [clientLog.channelEntryZ.parse({ channel: 1, color: color.ZERO })],
        channelNames: { "1": "log" },
        selectionStart: 0,
        selectionEnd: 0,
      });
      log.render();
      expect(selectedText(log)).toMatch(/\d{2}:\d{2}:\d{2} {2}log {3}v$/);
    });

    describe("timestamp channels", () => {
      const NANOS = BigInt(TimeStamp.seconds(1).valueOf());
      const copied = (channels: clientLog.ChannelEntry[]): string => {
        const entries = [
          { channelKey: 1, timestamp: TimeStamp.seconds(1), value: String(NANOS) },
        ];
        const { log } = setup(entries, REGION_500, {
          channels,
          channelDataTypes: { "1": "timestamp" },
          timestampsHidden: true,
          channelNamesHidden: true,
          selectionStart: 0,
          selectionEnd: 0,
        });
        log.render();
        return log.selection().lines[0].text;
      };

      it("should format the value with the channel's timestamp config", () => {
        const entry = clientLog.channelEntryZ.parse({
          channel: 1,
          color: color.ZERO,
          timestampConfig: { format: "ISO", timeZone: "UTC" },
        });
        expect(copied([entry])).toBe(new TimeStamp(NANOS).toString("ISO", "UTC"));
      });

      it("should use the default format for a channel with no entry", () => {
        expect(copied([])).toBe(new TimeStamp(NANOS).toString("preciseDate", "local"));
      });
    });

    it("should apply precision formatting when channelConfig has precision", () => {
      const entries: LogEntry[] = [
        {
          channelKey: 1,
          timestamp: TimeStamp.milliseconds(1000),
          value: "3.14159265",
        },
      ];
      const { log } = setup(entries, REGION_500, {
        channels: [
          clientLog.channelEntryZ.parse({
            channel: 1,
            color: color.ZERO,
            precision: 2,
          }),
        ],
        channelDataTypes: { "1": "float64" },
        selectionStart: 0,
        selectionEnd: 0,
      });
      log.render();
      expect(selectedText(log)).toContain("3.14");
    });

    it("should show the value as it arrives when the precision is absent", () => {
      const entries: LogEntry[] = [
        {
          channelKey: 1,
          timestamp: TimeStamp.milliseconds(1000),
          value: "3.14159265",
        },
      ];
      const { log } = setup(entries, REGION_500, {
        channels: [clientLog.channelEntryZ.parse({ channel: 1, color: color.ZERO })],
        selectionStart: 0,
        selectionEnd: 0,
      });
      log.render();
      expect(selectedText(log)).toContain("3.14159265");
    });

    it("should apply scientific notation when configured", () => {
      const entries: LogEntry[] = [
        {
          channelKey: 1,
          timestamp: TimeStamp.milliseconds(1000),
          value: "12345",
        },
      ];
      const { log } = setup(entries, REGION_500, {
        channels: [
          clientLog.channelEntryZ.parse({
            channel: 1,
            color: color.ZERO,
            notation: "scientific",
            precision: 2,
          }),
        ],
        channelDataTypes: { "1": "float64" },
        selectionStart: 0,
        selectionEnd: 0,
      });
      log.render();
      // Should contain scientific notation (uses unicode ᴇ)
      expect(selectedText(log)).toMatch(/[eEᴇ]/);
    });

    it("should render continuation entries without name/timestamp but aligned to the value column", () => {
      const entries: LogEntry[] = [
        { channelKey: 1, timestamp: TimeStamp.milliseconds(1000), value: "hello" },
        {
          channelKey: 1,
          timestamp: TimeStamp.milliseconds(1000),
          value: "world",
          continuation: true,
        },
        { channelKey: 1, timestamp: TimeStamp.milliseconds(2000), value: "again" },
      ];
      const { log } = setup(entries, REGION_500, {
        channelNamesHidden: false,
        timestampsHidden: true,
        channels: [clientLog.channelEntryZ.parse({ channel: 1, color: color.ZERO })],
        channelNames: { "1": "log" },
        selectionStart: 0,
        selectionEnd: 2,
      });
      log.render();
      const lines = log.selection().lines.map((l) => l.text);
      expect(lines[0]).toContain("log");
      expect(lines[0]).toContain("hello");
      // Continuation: no name, but leading whitespace preserves value alignment.
      expect(lines[1]).not.toContain("log");
      expect(lines[1]).toMatch(/^\s+world$/);
      expect(lines[1].indexOf("world")).toBe(lines[0].indexOf("hello"));
      expect(lines[2]).toContain("log");
      expect(lines[2]).toContain("again");
    });

    it("should render an entry with empty value but no continuation flag with its prefix intact", () => {
      const entries: LogEntry[] = [
        { channelKey: 1, timestamp: TimeStamp.milliseconds(1000), value: "" },
      ];
      const { log } = setup(entries, REGION_500, {
        channelNamesHidden: false,
        timestampsHidden: true,
        channels: [clientLog.channelEntryZ.parse({ channel: 1, color: color.ZERO })],
        channelNames: { "1": "log" },
        selectionStart: 0,
        selectionEnd: 0,
      });
      log.render();
      const [line] = log.selection().lines.map((l) => l.text);
      expect(line).toContain("log");
    });

    it("should render a three-line continuation group with consistent alignment", () => {
      const entries: LogEntry[] = [
        { channelKey: 1, timestamp: TimeStamp.milliseconds(1000), value: "first" },
        {
          channelKey: 1,
          timestamp: TimeStamp.milliseconds(1000),
          value: "second",
          continuation: true,
        },
        {
          channelKey: 1,
          timestamp: TimeStamp.milliseconds(1000),
          value: "third",
          continuation: true,
        },
      ];
      const { log } = setup(entries, REGION_500, {
        channelNamesHidden: false,
        timestampsHidden: false,
        channels: [clientLog.channelEntryZ.parse({ channel: 1, color: color.ZERO })],
        channelNames: { "1": "sensor" },
        selectionStart: 0,
        selectionEnd: 2,
      });
      log.render();
      const lines = log.selection().lines.map((l) => l.text);
      expect(lines[0]).toContain("sensor");
      expect(lines[0]).toContain("first");
      const valueCol = lines[0].indexOf("first");
      expect(lines[1]).not.toContain("sensor");
      expect(lines[1].indexOf("second")).toBe(valueCol);
      expect(lines[2]).not.toContain("sensor");
      expect(lines[2].indexOf("third")).toBe(valueCol);
    });

    it("should render continuation with both timestamp and channel name hidden", () => {
      const entries: LogEntry[] = [
        { channelKey: 1, timestamp: TimeStamp.milliseconds(1000), value: "line1" },
        {
          channelKey: 1,
          timestamp: TimeStamp.milliseconds(1000),
          value: "line2",
          continuation: true,
        },
      ];
      const { log } = setup(entries, REGION_500, {
        channelNamesHidden: true,
        timestampsHidden: true,
        channels: [clientLog.channelEntryZ.parse({ channel: 1, color: color.ZERO })],
        selectionStart: 0,
        selectionEnd: 1,
      });
      log.render();
      const lines = log.selection().lines.map((l) => l.text);
      expect(lines[0]).toBe("line1");
      expect(lines[1]).toBe("line2");
    });

    it("should render continuation entries interleaved between different channels", () => {
      const entries: LogEntry[] = [
        { channelKey: 1, timestamp: TimeStamp.milliseconds(1000), value: "a1" },
        {
          channelKey: 1,
          timestamp: TimeStamp.milliseconds(1000),
          value: "a2",
          continuation: true,
        },
        { channelKey: 2, timestamp: TimeStamp.milliseconds(1000), value: "b1" },
        {
          channelKey: 2,
          timestamp: TimeStamp.milliseconds(1000),
          value: "b2",
          continuation: true,
        },
      ];
      const { log } = setup(entries, REGION_500, {
        channelNamesHidden: false,
        timestampsHidden: true,
        channels: [
          clientLog.channelEntryZ.parse({ channel: 1, color: color.ZERO }),
          clientLog.channelEntryZ.parse({ channel: 2, color: color.ZERO }),
        ],
        channelNames: { "1": "chA", "2": "chB" },
        selectionStart: 0,
        selectionEnd: 3,
      });
      log.render();
      const lines = log.selection().lines.map((l) => l.text);
      expect(lines[0]).toContain("chA");
      expect(lines[0]).toContain("a1");
      expect(lines[1]).not.toContain("chA");
      expect(lines[1]).toContain("a2");
      expect(lines[2]).toContain("chB");
      expect(lines[2]).toContain("b1");
      expect(lines[3]).not.toContain("chB");
      expect(lines[3]).toContain("b2");
    });

    it("should render a continuation entry with an empty value as a blank aligned line", () => {
      const entries: LogEntry[] = [
        { channelKey: 1, timestamp: TimeStamp.milliseconds(1000), value: "data" },
        {
          channelKey: 1,
          timestamp: TimeStamp.milliseconds(1000),
          value: "",
          continuation: true,
        },
      ];
      const { log } = setup(entries, REGION_500, {
        channelNamesHidden: false,
        timestampsHidden: true,
        channels: [clientLog.channelEntryZ.parse({ channel: 1, color: color.ZERO })],
        channelNames: { "1": "log" },
        selectionStart: 0,
        selectionEnd: 1,
      });
      log.render();
      const lines = log.selection().lines.map((l) => l.text);
      expect(lines[0]).toContain("log");
      expect(lines[0]).toContain("data");
      expect(lines[1]).not.toContain("log");
      expect(lines[1]).toMatch(/^\s*$/);
      expect(lines[1].length).toBe(lines[0].indexOf("data"));
    });
  });

  describe("number formatting", () => {
    // Returns the value drawn for a sample of a channel with the given settings.
    const shown = (
      value: string,
      dataType: string | undefined,
      overrides: Partial<clientLog.ChannelEntry>,
    ): string => {
      const { log, recorder } = setup(
        [{ ...makeEntry(0), value }],
        box.construct({ x: 0, y: 0 }, { width: 1000, height: 500 }),
        {
          channels: [
            clientLog.channelEntryZ.parse({
              channel: 1,
              color: color.ZERO,
              ...overrides,
            }),
          ],
          channelDataTypes: dataType == null ? {} : { "1": dataType },
          channelNamesHidden: true,
          timestampsHidden: true,
        },
      );
      recorder.clear();
      log.render();
      return recorder.lower2d.calls
        .filter(({ op }) => op === "fillText")
        .map(({ args }) => String(args[0]))
        .join("");
    };

    it("should keep the text of a string channel with a precision", () => {
      expect(shown("12 volts", "string", { precision: 2 })).toBe("12 volts");
    });

    it("should keep the text of a string channel with a notation", () => {
      expect(shown("1234", "string", { notation: "scientific" })).toBe("1234");
    });

    it("should keep the value of a boolean channel with a precision", () => {
      expect(shown("true", "boolean", { precision: 2 })).toBe("true");
    });

    it("should keep a float sample of a channel that became an integer", () => {
      expect(shown("3.14", "int32", { precision: 2 })).toBe("3.14");
    });

    it("should keep text of a channel that became a float", () => {
      expect(shown("12 volts", "float32", { precision: 2 })).toBe("12 volts");
    });

    it("should keep a blank line of a channel that became a float", () => {
      expect(shown("  ", "float32", { precision: 2 })).toBe("  ");
    });

    it("should show the raw value before the data type loads", () => {
      expect(shown("3.14159", undefined, { precision: 2 })).toBe("3.14159");
    });

    it("should apply a precision to a float channel", () => {
      expect(shown("3.14159", "float64", { precision: 2 })).toBe("3.14");
    });

    it("should keep every digit at auto precision in scientific notation", () => {
      expect(shown("1234.5", "float64", { notation: "scientific" })).toBe("1.2345ᴇ3");
    });

    it("should keep every digit at auto precision in engineering notation", () => {
      expect(shown("12345.6", "float64", { notation: "engineering" })).toBe(
        "12.3456ᴇ3",
      );
    });

    it("should keep every digit of a large integer at a precision", () => {
      expect(shown("9007199254740993", "int64", { precision: 1 })).toBe(
        "9007199254740993.0",
      );
    });

    it("should keep every digit of a large integer at auto precision", () => {
      expect(shown("18446744073709551615", "uint64", { notation: "scientific" })).toBe(
        "1.8446744073709551615ᴇ19",
      );
    });
  });

  describe("time-based scrollbar", () => {
    const paused = (wheelPos: number, region: box.Box = REGION_500) => {
      const entries = Array.from({ length: 200 }, (_, i) => makeEntry(i));
      const { log, recorder, source, updateState } = setup(entries, region);
      source.timeRange = new TimeRange(
        TimeStamp.milliseconds(-1_000_000),
        entries[199].timestamp,
      );
      updateState({ scrolling: false, empty: false });
      updateState({ scrolling: true, empty: false, wheelPos: 0 });
      if (wheelPos !== 0) updateState({ scrolling: true, empty: false, wheelPos });
      recorder.clear();
      log.render();
      return findThumb(recorder);
    };

    it("should pin the thumb to the bottom at the newest time", () => {
      const thumb = paused(0);
      assert(thumb != null);
      expect(thumb.y + thumb.height).toBeCloseTo(box.bottom(REGION_500), 0);
    });

    it("should keep the thumb inside a track shorter than its minimum height", () => {
      const region = box.construct({ x: 0, y: 0 }, { width: 400, height: 30 });
      const thumb = paused(0, region);
      assert(thumb != null);
      expect(thumb.y).toBeGreaterThanOrEqual(0);
      expect(thumb.height).toBeLessThanOrEqual(30);
    });

    it("should keep the thumb when the view spans the times but hides lines", () => {
      const entries = Array.from({ length: 200 }, (_, i) => ({
        ...makeEntry(i),
        timestamp: TimeStamp.milliseconds(i === 199 ? 2_000 : 1_000),
      }));
      const { log, recorder, source, updateState } = setup(entries);
      source.timeRange = new TimeRange(entries[0].timestamp, entries[199].timestamp);
      updateState({ scrolling: false, empty: false });
      updateState({ scrolling: true, empty: false, wheelPos: 0 });
      recorder.clear();
      log.render();
      expect(findThumb(recorder)).not.toBeNull();
    });

    it("should sit by time, not by the lines held", () => {
      // The oldest entry held is most of the way down the stored times.
      const thumb = paused(100_000);
      assert(thumb != null);
      expect(thumb.y).toBeGreaterThan(box.height(REGION_500) * 0.75);
    });
  });

  describe("wide characters", () => {
    const rowsOf = (value: string): number => {
      const { log } = setup([{ ...makeEntry(0), value }]);
      log.render();
      return log.state.rowEntries.length;
    };

    it("should keep an empty value at every width", () => {
      for (let width = 20; width <= 160; width += 2) {
        const region = box.construct({ x: 0, y: 0 }, { width, height: 500 });
        const { log } = setup([{ ...makeEntry(0), value: "" }], region);
        log.render();
        expect(log.state.rowEntries).toContain(0);
      }
    });

    it("should give a wide character two columns", () => {
      const ascii = rowsOf("a".repeat(2_000));
      const wide = rowsOf("日".repeat(1_000));
      expect(Math.abs(wide - ascii)).toBeLessThanOrEqual(1);
    });

    it("should keep every character of a wide value", () => {
      const value = "日本語a".repeat(60);
      const { log, recorder } = setup([{ ...makeEntry(0), value }], REGION_500);
      recorder.clear();
      log.render();
      const drawn = recorder.lower2d.calls
        .filter(({ op }) => op === "fillText")
        .map(({ args }) => String(args[0]))
        .filter((text) => /[日本語]/.test(text));
      expect(drawn.length).toBeGreaterThan(1);
      expect(drawn.join("")).toBe(value);
    });
  });

  describe("scrolling through a wrapped value", () => {
    const LONG = Array.from({ length: 2_000 }, (_, i) => `w${i}`).join(" ");
    const pausedRows = (wheelPos: number) => {
      const entries = [makeEntry(0), { ...makeEntry(1), value: LONG }, makeEntry(2)];
      const { log, recorder, updateState } = setup(entries);
      updateState({ empty: false, wheelPos: 0 });
      updateState({ empty: false, wheelPos: 0, scrolling: true });
      for (let wheel = 0; wheel <= wheelPos; wheel += log.lineHeight)
        updateState({ empty: false, wheelPos: wheel, scrolling: true });
      recorder.clear();
      log.render();
      return {
        log,
        recorder,
        updateState,
        texts: recorder.lower2d.calls
          .filter(({ op }) => op === "fillText")
          .map(({ args }) => String(args[0])),
      };
    };

    it("should show the end of the value before scrolling", () => {
      const { texts } = pausedRows(0);
      expect(texts.some((t) => t.includes("w1999"))).toBe(true);
      expect(texts.some((t) => t.startsWith("w0 "))).toBe(false);
    });

    it("should reach the start of the value by scrolling row by row", () => {
      const { texts } = pausedRows(100_000);
      expect(texts.some((t) => t.startsWith("w0 "))).toBe(true);
    });

    it("should draw a thumb that moves through one value", () => {
      const thumbY = (wheelPos: number): number | undefined => {
        const { recorder } = pausedRows(wheelPos);
        return findThumb(recorder)?.y;
      };
      const [bottom, top] = [thumbY(0), thumbY(100_000)];
      expect(bottom).toBeDefined();
      expect(top).toBeLessThan(bottom!);
    });

    it("should keep scrolling after a resize shrinks the value", () => {
      const { log, updateState } = pausedRows(100_000);
      const { tail } = log.scrollState;
      expect(tail).toBeGreaterThan(0);
      log.entries[1] = { ...log.entries[1], value: "short" };
      updateState({
        empty: false,
        wheelPos: 100_000 + log.lineHeight,
        scrolling: true,
      });
      expect(log.scrollState.tail).toBe(0);
    });

    it("should show the entries of a log mounted paused", () => {
      const { log } = setup([makeEntry(0), makeEntry(1)], REGION_500, {
        scrolling: true,
      });
      log.render();
      expect(log.state.rowEntries).toContain(1);
    });

    it("should show entries that reach a log mounted paused", () => {
      const { log, source } = setup([], REGION_500, { scrolling: true });
      source.setEntries([makeEntry(0), makeEntry(1)]);
      source.notify();
      log.render();
      expect(log.state.rowEntries).toContain(1);
    });

    it("should reach the entry above the value", () => {
      const { log } = pausedRows(100_000);
      expect(log.state.rowEntries[0]).toBe(0);
    });
  });

  describe("days", () => {
    const entry = (date: string): LogEntry => ({
      channelKey: 1,
      timestamp: new TimeStamp(new Date(date)),
      value: "v",
    });
    const shown = (recorder: ReturnType<typeof canvasTest.record>): string[] =>
      recorder.lower2d.calls
        .filter(({ op }) => op === "fillText")
        .map(({ args }) => String(args[0]));

    it("should name the day above the first entry", () => {
      const { log, recorder } = setup([entry("2026-11-04T12:00:00")]);
      log.render();
      expect(shown(recorder)).toContain("Wed, Nov 4, 2026");
    });

    it("should show a day row where the day changes", () => {
      const entries = [entry("2026-11-03T23:00:00"), entry("2026-11-04T01:00:00")];
      const { log, recorder } = setup(entries);
      log.render();
      const texts = shown(recorder);
      expect(texts).toContain("Tue, Nov 3, 2026");
      expect(texts).toContain("Wed, Nov 4, 2026");
    });

    it("should show one day row for entries of one day", () => {
      const entries = [entry("2026-11-04T01:00:00"), entry("2026-11-04T23:00:00")];
      const { log, recorder } = setup(entries);
      log.render();
      expect(shown(recorder).filter((t) => t.includes("2026"))).toHaveLength(1);
    });

    describe("time zone", () => {
      it("should split day rows at midnight in UTC", () => {
        const entries = [entry("2026-11-03T23:00:00Z"), entry("2026-11-04T01:00:00Z")];
        const { log, recorder } = setup(entries, REGION_500, { timeZone: "UTC" });
        log.render();
        const texts = shown(recorder);
        expect(texts).toContain("Tue, Nov 3, 2026");
        expect(texts).toContain("Wed, Nov 4, 2026");
      });

      it("should show one day row for entries of one UTC day", () => {
        const entries = [entry("2026-11-04T01:00:00Z"), entry("2026-11-04T23:00:00Z")];
        const { log, recorder } = setup(entries, REGION_500, { timeZone: "UTC" });
        log.render();
        expect(shown(recorder).filter((t) => t.includes("2026"))).toEqual([
          "Wed, Nov 4, 2026",
        ]);
      });

      it("should show line times and copied dates in UTC", () => {
        const { log } = setup([entry("2026-01-05T01:02:03Z")], REGION_500, {
          timeZone: "UTC",
          selectionStart: 0,
          selectionEnd: 0,
        });
        log.render();
        expect(log.selection().lines[0].text).toMatch(/^2026-01-05 01:02:03/);
      });

      it("should default to local time", () => {
        const { log } = setup([entry("2026-01-05T01:02:03")], REGION_500, {
          selectionStart: 0,
          selectionEnd: 0,
        });
        log.render();
        expect(log.state.timeZone).toBe("local");
        expect(log.selection().lines[0].text).toMatch(/^2026-01-05 01:02:03/);
      });
    });

    it("should show no day rows when timestamps are hidden", () => {
      const { log, recorder } = setup([entry("2026-11-04T12:00:00")], REGION_500, {
        timestampsHidden: true,
      });
      log.render();
      expect(shown(recorder).filter((t) => t.includes("2026"))).toHaveLength(0);
    });

    it("should keep the date on copied lines", () => {
      const entries = [entry("2026-01-05T01:00:00"), entry("2026-01-05T23:00:00")];
      const { log } = setup(entries, REGION_500, {
        selectionStart: 0,
        selectionEnd: 1,
      });
      log.render();
      expect(log.selection().lines[0].text).toMatch(/^2026-01-05 01:00:00/);
    });

    it("should keep dates off copied lines when timestamps are hidden", () => {
      const { log } = setup([entry("2026-01-05T12:00:00")], REGION_500, {
        timestampsHidden: true,
        channelNamesHidden: true,
        selectionStart: 0,
        selectionEnd: 0,
      });
      log.render();
      expect(log.selection().lines[0].text).toBe("v");
    });
  });

  describe("per-frame selection cost", () => {
    it("should cover entries that arrive after a select-all", () => {
      const { log, source, updateState } = setup([makeEntry(0)]);
      updateState({ selectionStart: 0, selectionEnd: SELECT_ALL_END });
      source.push(makeEntry(1));
      source.notify();
      expect(log.selection().lines).toHaveLength(2);
    });

    const countReads = (count: number) => {
      const reads = { count: 0 };
      const entries = Array.from({ length: count }, (_, i) =>
        Object.defineProperty(makeEntry(i), "value", {
          get: () => {
            reads.count++;
            return String(i);
          },
        }),
      );
      return { reads, entries };
    };

    it("should not read the selected entries off screen on a frame", () => {
      const small = countReads(1_000);
      const large = countReads(20_000);
      const frameReads = ({ reads, entries }: typeof small): number => {
        const { log } = setup(entries, REGION_500, {
          selectionStart: 0,
          selectionEnd: SELECT_ALL_END,
        });
        reads.count = 0;
        log.render();
        return reads.count;
      };
      expect(frameReads(large)).toBe(frameReads(small));
    });
  });

  describe("selection", () => {
    it("should be empty when selection is negative", () => {
      const entries = Array.from({ length: 5 }, (_, i) => makeEntry(i));
      const { log } = setup(entries, REGION_500, {
        selectionStart: -1,
        selectionEnd: -1,
      });
      log.render();
      expect(selectedText(log)).toBe("");
      expect(log.selection().lines).toEqual([]);
    });

    it("should return a single selected entry", () => {
      const entries = Array.from({ length: 10 }, (_, i) => makeEntry(i));
      const { log } = setup(entries, REGION_500, {
        selectionStart: 0,
        selectionEnd: 0,
      });
      log.render();
      expect(selectedText(log).length).toBeGreaterThan(0);
      expect(log.selection().lines).toHaveLength(1);
    });

    it("should return a range of selected entries", () => {
      const entries = Array.from({ length: 10 }, (_, i) => makeEntry(i));
      const { log } = setup(entries, REGION_500, {
        selectionStart: 2,
        selectionEnd: 5,
      });
      log.render();
      expect(log.selection().lines).toHaveLength(4);
      expect(selectedText(log)).toContain("\n");
    });

    it("should clamp a select-to-end sentinel to all entries", () => {
      const entries = Array.from({ length: 10 }, (_, i) => makeEntry(i));
      const { log } = setup(entries, REGION_500, {
        selectionStart: 0,
        selectionEnd: Number.MAX_SAFE_INTEGER,
      });
      log.render();
      expect(log.selection().lines).toHaveLength(10);
    });

    it("should handle reversed selection (end < start)", () => {
      const entries = Array.from({ length: 10 }, (_, i) => makeEntry(i));
      const { log } = setup(entries, REGION_500, {
        selectionStart: 5,
        selectionEnd: 2,
      });
      log.render();
      expect(log.selection().lines).toHaveLength(4);
    });

    it("should include the color of a channel with a custom color", () => {
      const entries = Array.from({ length: 5 }, (_, i) => makeEntry(i));
      const { log } = setup(entries, REGION_500, {
        channels: [
          clientLog.channelEntryZ.parse({
            channel: 1,
            color: color.construct("#ff0000"),
          }),
        ],
        selectionStart: 0,
        selectionEnd: 0,
      });
      log.render();
      expect(log.selection().lines[0].color).not.toBe("");
    });

    it("should have an empty color when a channel has no custom color", () => {
      const entries = Array.from({ length: 5 }, (_, i) => makeEntry(i));
      const { log } = setup(entries, REGION_500, {
        selectionStart: 0,
        selectionEnd: 0,
      });
      log.render();
      expect(log.selection().lines[0].color).toBe("");
    });
  });

  describe("selection partial", () => {
    const entries = Array.from({ length: 5 }, (_, i) => makeEntry(i));
    const SELECT_ALL = { selectionStart: 0, selectionEnd: SELECT_ALL_END };

    it("should be true for a select-all when older values are stored", () => {
      const { log } = setup(entries, REGION_500, SELECT_ALL);
      expect(log.selection().partial).toBe(true);
    });

    it("should be true for a select-all when newer values are stored", () => {
      const { log, source, updateState } = setup(entries, REGION_500, SELECT_ALL);
      source.exhausted = true;
      source.live = false;
      updateState(SELECT_ALL);
      expect(log.selection().partial).toBe(true);
    });

    it("should be false for a select-all of a log that holds every value", () => {
      const { log, source, updateState } = setup(entries, REGION_500, SELECT_ALL);
      source.exhausted = true;
      updateState(SELECT_ALL);
      expect(log.selection().partial).toBe(false);
    });

    it("should be false for a selection that is not a select-all", () => {
      const { log } = setup(entries, REGION_500, {
        selectionStart: 0,
        selectionEnd: 4,
      });
      expect(log.selection().partial).toBe(false);
    });
  });

  describe("render with selection highlighting", () => {
    it("should not render selection when no selection is active", () => {
      const entries = Array.from({ length: 10 }, (_, i) => makeEntry(i));
      const { log } = setup(entries, REGION_500, {
        selectionStart: -1,
        selectionEnd: -1,
      });
      const result = log.render();
      expect(result).toBeTypeOf("function");
    });

    it("should render selection highlight when selection is within visible range", () => {
      const entries = Array.from({ length: 10 }, (_, i) => makeEntry(i));
      const { log } = setup(entries, REGION_500, {
        selectionStart: 2,
        selectionEnd: 5,
      });
      const result = log.render();
      expect(result).toBeTypeOf("function");
    });
  });

  describe("render scrollbar", () => {
    // The log never sits at the top left of the window, and the thumb is positioned in
    // canvas coordinates, so an offset region is the case that matters.
    const REGION_OFFSET = box.construct({ x: 20, y: 200 }, { width: 400, height: 500 });

    // Enters scrollback pinned to the bottom, then scrolls up by wheelPos.
    const scrollTo = (
      updateState: (o: Record<string, unknown>) => void,
      wheelPos: number,
    ): void => {
      updateState({ scrolling: false, empty: false });
      updateState({ scrolling: true, empty: false, wheelPos: 0 });
      if (wheelPos !== 0) updateState({ scrolling: true, empty: false, wheelPos });
    };

    it("should not draw a thumb while the log is live", () => {
      const entries = Array.from({ length: 500 }, (_, i) => makeEntry(i));
      const { log, recorder } = setup(entries);
      log.render();
      expect(findThumb(recorder)).toBeNull();
    });

    it("should not draw a thumb when the log has no entries", () => {
      const { log, recorder, updateState } = setup([]);
      updateState({ scrolling: true, empty: true });
      recorder.clear();
      log.render();
      expect(findThumb(recorder)).toBeNull();
    });

    it("should not draw a thumb when the content barely overflows", () => {
      const { log, recorder, updateState } = setup(
        Array.from({ length: 33 }, (_, i) => makeEntry(i)),
      );
      scrollTo(updateState, 0);
      recorder.clear();
      log.render();
      // 33 lines at a 15px line height is 495px against a 500px region.
      expect(log.totalHeight).toBeLessThan(box.height(REGION_500));
      expect(findThumb(recorder)).toBeNull();
    });

    it("should size the thumb to the visible fraction of the content", () => {
      const entries = Array.from({ length: 200 }, (_, i) => makeEntry(i));
      const { log, recorder, updateState } = setup(entries);
      scrollTo(updateState, 0);
      recorder.clear();
      log.render();
      const regHeight = box.height(REGION_500);
      const thumb = findThumb(recorder);
      assert(thumb != null);
      // The day row takes one row of the view.
      const shown = (log.visibleLineCount - 1) / log.visibleLineCount;
      expect(thumb.height).toBeCloseTo(
        (regHeight / log.totalHeight) * regHeight * shown,
      );
    });

    it("should floor the thumb height when the content dwarfs the region", () => {
      const entries = Array.from({ length: 5000 }, (_, i) => makeEntry(i));
      const { log, recorder, updateState } = setup(entries);
      scrollTo(updateState, 0);
      recorder.clear();
      log.render();
      const regHeight = box.height(REGION_500);
      expect((regHeight / log.totalHeight) * regHeight).toBeLessThan(32);
      const thumb = findThumb(recorder);
      assert(thumb != null);
      expect(thumb.height).toBe(32);
    });

    it("should pin the thumb to the bottom of the region at the newest entry", () => {
      const entries = Array.from({ length: 200 }, (_, i) => makeEntry(i));
      const { log, recorder, updateState } = setup(entries, REGION_OFFSET);
      scrollTo(updateState, 0);
      recorder.clear();
      log.render();
      const thumb = findThumb(recorder);
      assert(thumb != null);
      expect(thumb.y + thumb.height).toBeCloseTo(box.bottom(REGION_OFFSET));
      expect(thumb.x + thumb.width).toBeCloseTo(box.right(REGION_OFFSET));
    });

    it("should pin the thumb to the top of the region at the oldest entry", () => {
      const entries = Array.from({ length: 200 }, (_, i) => makeEntry(i));
      const { log, recorder, updateState } = setup(entries, REGION_OFFSET);
      scrollTo(updateState, 100_000);
      recorder.clear();
      log.render();
      expect(log.scrollState.offset).toBe(log.visibleLineCount - 1);
      const thumb = findThumb(recorder);
      assert(thumb != null);
      expect(thumb.y).toBeCloseTo(box.top(REGION_OFFSET));
    });

    it("should keep the thumb inside the region while scrolling up", () => {
      const entries = Array.from({ length: 200 }, (_, i) => makeEntry(i));
      const { log, recorder, updateState } = setup(entries, REGION_OFFSET);
      updateState({ scrolling: false, empty: false });
      updateState({ scrolling: true, empty: false, wheelPos: 0 });
      for (let wheelPos = 100; wheelPos <= 5000; wheelPos += 100) {
        updateState({ scrolling: true, empty: false, wheelPos });
        recorder.clear();
        log.render();
        const thumb = findThumb(recorder);
        assert(thumb != null);
        expect(thumb.y).toBeGreaterThanOrEqual(box.top(REGION_OFFSET));
        expect(thumb.y + thumb.height).toBeLessThanOrEqual(box.bottom(REGION_OFFSET));
      }
    });
  });

  describe("multi-channel entries", () => {
    it("should handle entries from multiple channels", () => {
      const entries = [
        makeEntry(0, 1),
        makeEntry(1, 2),
        makeEntry(2, 1),
        makeEntry(3, 3),
      ];
      const { log } = setup(entries, REGION_500, {
        channelNamesHidden: false,
        channels: [
          clientLog.channelEntryZ.parse({ channel: 1, color: color.ZERO }),
          clientLog.channelEntryZ.parse({ channel: 2, color: color.ZERO }),
          clientLog.channelEntryZ.parse({ channel: 3, color: color.ZERO }),
        ],
        channelNames: { "1": "Temperature", "2": "Pressure", "3": "Humidity" },
        selectionStart: 0,
        selectionEnd: 3,
      });
      log.render();
      expect(log.selection().lines).toHaveLength(4);
      expect(selectedText(log)).toContain("Temperature");
      expect(selectedText(log)).toContain("Pressure");
      expect(selectedText(log)).toContain("Humidity");
    });

    it("should apply per-channel colors from channelConfigs", () => {
      const entries = [makeEntry(0, 1), makeEntry(1, 2)];
      const { log } = setup(entries, REGION_500, {
        channels: [
          clientLog.channelEntryZ.parse({
            channel: 1,
            color: color.construct("#ff0000"),
          }),
          clientLog.channelEntryZ.parse({
            channel: 2,
            color: color.construct("#00ff00"),
          }),
        ],
        selectionStart: 0,
        selectionEnd: 1,
      });
      log.render();
      expect(log.selection().lines[0].color).not.toBe("");
      expect(log.selection().lines[1].color).not.toBe("");
      expect(log.selection().lines[0].color).not.toBe(log.selection().lines[1].color);
    });
  });

  describe("theme handling", () => {
    it("should work with light theme", () => {
      const lightTheme = theme.themeZ.parse(theme.SYNNAX_LIGHT);
      const entries = Array.from({ length: 5 }, (_, i) => makeEntry(i));
      const { log } = setup(entries, REGION_500, {}, lightTheme);
      expect(log.lineHeight).toBeGreaterThan(0);
      expect(log.entries).toHaveLength(5);
    });
  });

  describe("afterDelete", () => {
    it("should clean up telem", () => {
      const { h, source } = setup([makeEntry(0)]);
      const cleanupSpy = vi.spyOn(source, "cleanup");
      h.unmount();
      expect(cleanupSpy).toHaveBeenCalled();
    });

    it("should erase the last drawn region, and draw no render left pending", () => {
      vi.useFakeTimers({ toFake: ["requestAnimationFrame"] });
      const { h, recorder } = setup([makeEntry(0)]);
      const loop = new render.Loop({ handleError: vi.fn() });
      recorder.loop.set = (req) => loop.set(req);
      h.setState((s) => ({ ...s }));
      vi.advanceTimersToNextFrame();
      h.setState((s) => ({ ...s, region: box.translate(s.region, "x", 100) }));
      recorder.clear();
      h.unmount();
      vi.advanceTimersToNextFrame();
      vi.useRealTimers();
      expect(recorder.eraseCalls.map(({ region }) => region)).toEqual([REGION_500]);
      expect(recorder.lower2d.calls).toEqual([]);
    });
  });
});
