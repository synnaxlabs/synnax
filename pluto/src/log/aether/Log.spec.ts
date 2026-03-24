// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { box, color, TimeStamp } from "@synnaxlabs/x";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { channelConfigZ, Log, logState } from "@/log/aether/Log";
import { type LogEntry } from "@/log/aether/types";
import { SYNNAX_DARK, SYNNAX_LIGHT, type Theme, themeZ } from "@/theming/base/theme";

const MockSender = { send: vi.fn() };

const THEME: Theme = themeZ.parse(SYNNAX_DARK);

const mockLogSource = (entries: LogEntry[] = []) => ({
  value: vi.fn(() => entries),
  evictedCount: 0,
  cleanup: vi.fn(),
  onChange: vi.fn((_cb: () => void) => () => {}),
  setChannels: vi.fn(),
});

const mockCanvas2DContext = () => ({
  setLineDash: vi.fn(),
  beginPath: vi.fn(),
  closePath: vi.fn(),
  stroke: vi.fn(),
  fill: vi.fn(),
  rect: vi.fn(),
  roundRect: vi.fn(),
  fillText: vi.fn(),
  strokeRect: vi.fn(),
  fillRect: vi.fn(),
  save: vi.fn(),
  restore: vi.fn(),
  clip: vi.fn(),
  clearRect: vi.fn(),
  moveTo: vi.fn(),
  lineTo: vi.fn(),
  arc: vi.fn(),
  fillStyle: "",
  strokeStyle: "",
  lineWidth: 1,
  font: "",
  measureText: vi.fn(() => ({ width: 8 })),
});

const mockRenderContext = () => {
  const ctx2d = mockCanvas2DContext();
  return {
    loop: { set: vi.fn() },
    erase: vi.fn(),
    scissor: vi.fn(() => vi.fn()),
    lower2d: {
      canvas: { width: 800, height: 600 },
      getContext: vi.fn(() => ctx2d),
      ...ctx2d,
      font: "",
      scissor: vi.fn(() => vi.fn()),
    },
  };
};

const mockTelemContext = (source: ReturnType<typeof mockLogSource>) => ({
  key: "test-telem-ctx",
  create: vi.fn(() => source),
  child: vi.fn(),
});

const createLog = (parentCtx?: Map<string, unknown>) => {
  const ctx = parentCtx ?? new Map<string, unknown>();
  return new Log({
    key: "test-log",
    type: "log",
    sender: MockSender,
    instrumentation: alamos.Instrumentation.NOOP,
    parentCtxValues: ctx,
  });
};

const REGION_500 = box.construct({ x: 0, y: 0 }, { width: 400, height: 500 });

const makeEntry = (i: number, channelKey: number = 1): LogEntry => ({
  channelKey,
  timestamp: BigInt(TimeStamp.milliseconds(i * 1000).valueOf()),
  value: String(i),
});

const setupWithContext = (
  entries: LogEntry[] = [],
  region: box.Box = REGION_500,
  stateOverrides: Record<string, unknown> = {},
  theme: Theme = THEME,
): {
  log: Log;
  source: ReturnType<typeof mockLogSource>;
  renderCtx: ReturnType<typeof mockRenderContext>;
} => {
  const source = mockLogSource(entries);
  const renderCtx = mockRenderContext();
  const telemCtx = mockTelemContext(source);
  const parentCtx = new Map<string, unknown>([
    ["pluto-theming-context", theme],
    ["pluto-render-context", renderCtx],
    ["pluto-telem-context", telemCtx],
  ]);
  const log = createLog(parentCtx);
  const state = logState.parse({
    region,
    wheelPos: 0,
    scrolling: false,
    empty: true,
    visible: true,
    ...stateOverrides,
  });
  log._updateState({
    path: ["test-log"],
    state,
    type: "log",
    create: () => log,
  });
  return { log, source, renderCtx };
};

describe("log/aether/Log", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("lineHeight", () => {
    it("should calculate from theme typography and base size", () => {
      const { log } = setupWithContext();
      const expected = THEME.typography[log.state.font].size * THEME.sizes.base;
      expect(log.lineHeight).toBe(expected);
      expect(log.lineHeight).toBeGreaterThan(0);
    });
  });

  describe("visibleLineCount", () => {
    it("should calculate how many lines fit in the region", () => {
      const entries = Array.from({ length: 50 }, (_, i) => makeEntry(i));
      const { log } = setupWithContext(entries);
      const regionHeight = box.height(REGION_500);
      const expected = Math.min(
        Math.floor((regionHeight - 12) / log.lineHeight),
        entries.length,
      );
      expect(log.visibleLineCount).toBe(expected);
    });

    it("should be capped at entry count when fewer entries than viewport", () => {
      const entries = [makeEntry(0)];
      const { log } = setupWithContext(entries);
      expect(log.visibleLineCount).toBe(1);
    });
  });

  describe("totalHeight", () => {
    it("should be entries.length * lineHeight", () => {
      const entries = Array.from({ length: 10 }, (_, i) => makeEntry(i));
      const { log } = setupWithContext(entries);
      expect(log.totalHeight).toBeCloseTo(entries.length * log.lineHeight);
    });
  });

  describe("empty state detection", () => {
    it("should set empty to false when entries arrive", () => {
      const entries = [makeEntry(0)];
      const { log } = setupWithContext(entries);
      expect(log.state.empty).toBe(false);
    });

    it("should keep empty true when no entries", () => {
      const { log } = setupWithContext([]);
      expect(log.state.empty).toBe(true);
    });
  });

  describe("scrollback", () => {
    it("should initialize scrollState with zero values", () => {
      const { log } = setupWithContext();
      expect(log.scrollState.offset).toBe(0);
      expect(log.scrollState.offsetRef).toBe(0);
      expect(log.scrollState.scrollRef).toBe(0);
    });

    it("should enter scrollback when scrolling transitions from false to true", () => {
      const entries = Array.from({ length: 100 }, (_, i) => makeEntry(i));
      const source = mockLogSource(entries);
      const renderCtx = mockRenderContext();
      const telemCtx = mockTelemContext(source);
      const parentCtx = new Map<string, unknown>([
        ["pluto-theming-context", THEME],
        ["pluto-render-context", renderCtx],
        ["pluto-telem-context", telemCtx],
      ]);
      const log = createLog(parentCtx);

      // First update: not scrolling
      log._updateState({
        path: ["test-log"],
        state: logState.parse({
          region: REGION_500,
          wheelPos: 0,
          scrolling: false,
          empty: true,
          visible: true,
        }),
        type: "log",
        create: () => log,
      });

      // Second update: start scrolling
      log._updateState({
        path: ["test-log"],
        state: logState.parse({
          region: REGION_500,
          wheelPos: 100,
          scrolling: true,
          empty: false,
          visible: true,
        }),
        type: "log",
        create: () => log,
      });

      // Scrollback should be initialized with current entry count
      expect(log.scrollState.offset).toBe(entries.length);
      expect(log.scrollState.offsetRef).toBe(entries.length);
      expect(log.scrollState.scrollRef).toBe(100);
    });
  });

  describe("entries", () => {
    it("should store entries from telem source", () => {
      const entries = Array.from({ length: 5 }, (_, i) => makeEntry(i));
      const { log } = setupWithContext(entries);
      expect(log.entries).toHaveLength(5);
      expect(log.entries[0].value).toBe("0");
      expect(log.entries[4].value).toBe("4");
    });
  });

  describe("channelConfigZ", () => {
    it("should parse with defaults", () => {
      const parsed = channelConfigZ.parse({});
      expect(parsed.color).toBe("");
      expect(parsed.notation).toBe("standard");
      expect(parsed.precision).toBe(-1);
      expect(parsed.alias).toBe("");
    });

    it("should accept valid values", () => {
      const parsed = channelConfigZ.parse({
        color: "#ff0000",
        notation: "scientific",
        precision: 5,
        alias: "Temperature",
      });
      expect(parsed.color).toBe("#ff0000");
      expect(parsed.notation).toBe("scientific");
      expect(parsed.precision).toBe(5);
      expect(parsed.alias).toBe("Temperature");
    });

    it("should reject precision above 17", () => {
      expect(() => channelConfigZ.parse({ precision: 18 })).toThrow();
    });

    it("should reject precision below -1", () => {
      expect(() => channelConfigZ.parse({ precision: -2 })).toThrow();
    });
  });

  describe("logState schema", () => {
    it("should provide defaults for new selection fields", () => {
      const parsed = logState.parse({
        region: REGION_500,
        wheelPos: 0,
        scrolling: false,
        empty: true,
        visible: true,
      });
      expect(parsed.selectionStart).toBe(-1);
      expect(parsed.selectionEnd).toBe(-1);
      expect(parsed.selectedText).toBe("");
      expect(parsed.selectedLines).toEqual([]);
      expect(parsed.computedLineHeight).toBe(0);
      expect(parsed.entryCount).toBe(0);
      expect(parsed.copyFlash).toBe(false);
    });

    it("should provide defaults for channel-related fields", () => {
      const parsed = logState.parse({
        region: REGION_500,
        wheelPos: 0,
        scrolling: false,
        empty: true,
        visible: true,
      });
      expect(parsed.showChannelNames).toBe(true);
      expect(parsed.timestampPrecision).toBe(0);
      expect(parsed.channelNames).toEqual({});
      expect(parsed.channels).toEqual([]);
    });

    it("should accept explicit channel config values", () => {
      const parsed = logState.parse({
        region: REGION_500,
        wheelPos: 0,
        scrolling: false,
        empty: true,
        visible: true,
        channels: [{ channel: 1, color: "#ff0000", precision: 3 }, { channel: 2 }],
        showChannelNames: false,
        timestampPrecision: 2,
      });
      expect(parsed.showChannelNames).toBe(false);
      expect(parsed.timestampPrecision).toBe(2);
      expect(parsed.channels).toHaveLength(2);
      expect(parsed.channels[0].color).toBe("#ff0000");
      expect(parsed.channels[0].precision).toBe(3);
    });
  });

  describe("entryCount tracking", () => {
    it("should set entryCount when entries arrive", () => {
      const entries = Array.from({ length: 7 }, (_, i) => makeEntry(i));
      const { log } = setupWithContext(entries);
      expect(log.state.entryCount).toBe(7);
    });

    it("should keep entryCount at 0 when no entries", () => {
      const { log } = setupWithContext([]);
      expect(log.state.entryCount).toBe(0);
    });
  });

  describe("timestampPrecision", () => {
    it("should default to 0 and produce 8-char timestamps", () => {
      const { log } = setupWithContext();
      expect(log.state.timestampPrecision).toBe(0);
    });

    it("should accept precision values 0-3", () => {
      const { log } = setupWithContext([], REGION_500, { timestampPrecision: 3 });
      expect(log.state.timestampPrecision).toBe(3);
    });
  });

  describe("channel management", () => {
    it("should call setChannels on telem source", () => {
      const entries = [makeEntry(0)];
      const { source } = setupWithContext(entries, REGION_500, {
        channels: [{ channel: 1 }, { channel: 2 }, { channel: 3 }],
      });
      expect(source.setChannels).toHaveBeenCalledWith([1, 2, 3]);
    });

  });

  describe("color handling", () => {
    it("should use gray.l11 as text color when no custom color is set", () => {
      const { log } = setupWithContext([makeEntry(0)]);
      expect(color.isZero(log.state.color)).toBe(true);
    });

    it("should accept a custom text color", () => {
      const { log } = setupWithContext([], REGION_500, { color: "#ff0000" });
      expect(color.isZero(log.state.color)).toBe(false);
    });
  });

  describe("visibility", () => {
    it("should request render when visible", () => {
      const { renderCtx } = setupWithContext([makeEntry(0)], REGION_500, {
        visible: true,
      });
      expect(renderCtx.loop.set).toHaveBeenCalled();
    });

    it("should skip render when not visible and prevState also not visible", () => {
      const { renderCtx } = setupWithContext([makeEntry(0)], REGION_500, {
        visible: false,
      });
      // On the very first update, both state and prevState have visible=false,
      // so the early return on line 217 fires and no render is requested.
      expect(renderCtx.loop.set).not.toHaveBeenCalled();
    });
  });

  describe("scrollback with continued scrolling", () => {
    it("should adjust offset based on wheel position delta", () => {
      const entries = Array.from({ length: 100 }, (_, i) => makeEntry(i));
      const source = mockLogSource(entries);
      const renderCtx = mockRenderContext();
      const telemCtx = mockTelemContext(source);
      const parentCtx = new Map<string, unknown>([
        ["pluto-theming-context", THEME],
        ["pluto-render-context", renderCtx],
        ["pluto-telem-context", telemCtx],
      ]);
      const log = createLog(parentCtx);

      // First update: not scrolling
      log._updateState({
        path: ["test-log"],
        state: logState.parse({
          region: REGION_500,
          wheelPos: 0,
          scrolling: false,
          empty: true,
          visible: true,
        }),
        type: "log",
        create: () => log,
      });

      // Enter scrollback
      log._updateState({
        path: ["test-log"],
        state: logState.parse({
          region: REGION_500,
          wheelPos: 100,
          scrolling: true,
          empty: false,
          visible: true,
        }),
        type: "log",
        create: () => log,
      });

      const initialOffset = log.scrollState.offset;
      expect(initialOffset).toBe(entries.length);

      // Continue scrolling (wheel position changes)
      log._updateState({
        path: ["test-log"],
        state: logState.parse({
          region: REGION_500,
          wheelPos: 200,
          scrolling: true,
          empty: false,
          visible: true,
        }),
        type: "log",
        create: () => log,
      });

      // Offset should have changed based on the wheel delta
      expect(log.scrollState.offset).toBeLessThanOrEqual(entries.length);
      expect(log.scrollState.offset).toBeGreaterThan(0);
    });

    it("should exit scrollback when offset reaches entry count", () => {
      const entries = Array.from({ length: 100 }, (_, i) => makeEntry(i));
      const source = mockLogSource(entries);
      const renderCtx = mockRenderContext();
      const telemCtx = mockTelemContext(source);
      const parentCtx = new Map<string, unknown>([
        ["pluto-theming-context", THEME],
        ["pluto-render-context", renderCtx],
        ["pluto-telem-context", telemCtx],
      ]);
      const log = createLog(parentCtx);

      log._updateState({
        path: ["test-log"],
        state: logState.parse({
          region: REGION_500,
          wheelPos: 0,
          scrolling: false,
          empty: true,
          visible: true,
        }),
        type: "log",
        create: () => log,
      });

      log._updateState({
        path: ["test-log"],
        state: logState.parse({
          region: REGION_500,
          wheelPos: 100,
          scrolling: true,
          empty: false,
          visible: true,
        }),
        type: "log",
        create: () => log,
      });

      // Scroll back down past the end (large negative delta from scrollRef)
      log._updateState({
        path: ["test-log"],
        state: logState.parse({
          region: REGION_500,
          wheelPos: -5000,
          scrolling: true,
          empty: false,
          visible: true,
        }),
        type: "log",
        create: () => log,
      });

      // Should have exited scrollback
      expect(log.state.scrolling).toBe(false);
    });
  });

  describe("selection clamping on eviction", () => {
    it("should adjust selection indices when entries are evicted", () => {
      const entries = Array.from({ length: 20 }, (_, i) => makeEntry(i));
      const source = mockLogSource(entries);
      const renderCtx = mockRenderContext();
      const telemCtx = mockTelemContext(source);
      const parentCtx = new Map<string, unknown>([
        ["pluto-theming-context", THEME],
        ["pluto-render-context", renderCtx],
        ["pluto-telem-context", telemCtx],
      ]);
      const log = createLog(parentCtx);

      log._updateState({
        path: ["test-log"],
        state: logState.parse({
          region: REGION_500,
          wheelPos: 0,
          scrolling: false,
          empty: true,
          visible: true,
          selectionStart: 5,
          selectionEnd: 10,
        }),
        type: "log",
        create: () => log,
      });

      // Simulate eviction via the onChange callback
      const onChangeCallback = source.onChange.mock.calls[0][0];
      source.evictedCount = 3;
      source.value.mockReturnValue(entries.slice(3));
      onChangeCallback();

      // Selection should be adjusted by evictedCount
      expect(log.state.selectionStart).toBe(2);
      expect(log.state.selectionEnd).toBe(7);
    });

    it("should clear selection when all selected entries are evicted", () => {
      const entries = Array.from({ length: 20 }, (_, i) => makeEntry(i));
      const source = mockLogSource(entries);
      const renderCtx = mockRenderContext();
      const telemCtx = mockTelemContext(source);
      const parentCtx = new Map<string, unknown>([
        ["pluto-theming-context", THEME],
        ["pluto-render-context", renderCtx],
        ["pluto-telem-context", telemCtx],
      ]);
      const log = createLog(parentCtx);

      log._updateState({
        path: ["test-log"],
        state: logState.parse({
          region: REGION_500,
          wheelPos: 0,
          scrolling: false,
          empty: true,
          visible: true,
          selectionStart: 0,
          selectionEnd: 2,
        }),
        type: "log",
        create: () => log,
      });

      // Evict more than the selection range
      const onChangeCallback = source.onChange.mock.calls[0][0];
      source.evictedCount = 5;
      source.value.mockReturnValue(entries.slice(5));
      onChangeCallback();

      expect(log.state.selectionStart).toBe(-1);
      expect(log.state.selectionEnd).toBe(-1);
      expect(log.state.selectedText).toBe("");
    });

    it("should not modify selection when no entries are evicted", () => {
      const entries = Array.from({ length: 20 }, (_, i) => makeEntry(i));
      const source = mockLogSource(entries);
      const renderCtx = mockRenderContext();
      const telemCtx = mockTelemContext(source);
      const parentCtx = new Map<string, unknown>([
        ["pluto-theming-context", THEME],
        ["pluto-render-context", renderCtx],
        ["pluto-telem-context", telemCtx],
      ]);
      const log = createLog(parentCtx);

      log._updateState({
        path: ["test-log"],
        state: logState.parse({
          region: REGION_500,
          wheelPos: 0,
          scrolling: false,
          empty: true,
          visible: true,
          selectionStart: 5,
          selectionEnd: 10,
        }),
        type: "log",
        create: () => log,
      });

      // No eviction
      const onChangeCallback = source.onChange.mock.calls[0][0];
      source.evictedCount = 0;
      onChangeCallback();

      expect(log.state.selectionStart).toBe(5);
      expect(log.state.selectionEnd).toBe(10);
    });

    it("should clamp selectionStart to 0 when partially evicted", () => {
      const entries = Array.from({ length: 20 }, (_, i) => makeEntry(i));
      const source = mockLogSource(entries);
      const renderCtx = mockRenderContext();
      const telemCtx = mockTelemContext(source);
      const parentCtx = new Map<string, unknown>([
        ["pluto-theming-context", THEME],
        ["pluto-render-context", renderCtx],
        ["pluto-telem-context", telemCtx],
      ]);
      const log = createLog(parentCtx);

      log._updateState({
        path: ["test-log"],
        state: logState.parse({
          region: REGION_500,
          wheelPos: 0,
          scrolling: false,
          empty: true,
          visible: true,
          selectionStart: 2,
          selectionEnd: 8,
        }),
        type: "log",
        create: () => log,
      });

      // Evict 4 entries — start goes negative, end stays positive
      const onChangeCallback = source.onChange.mock.calls[0][0];
      source.evictedCount = 4;
      source.value.mockReturnValue(entries.slice(4));
      onChangeCallback();

      expect(log.state.selectionStart).toBe(0);
      expect(log.state.selectionEnd).toBe(4);
    });
  });

  describe("scrollback offset adjustment on eviction", () => {
    it("should reduce scroll offset when entries are evicted during scrollback", () => {
      const entries = Array.from({ length: 100 }, (_, i) => makeEntry(i));
      const source = mockLogSource(entries);
      const renderCtx = mockRenderContext();
      const telemCtx = mockTelemContext(source);
      const parentCtx = new Map<string, unknown>([
        ["pluto-theming-context", THEME],
        ["pluto-render-context", renderCtx],
        ["pluto-telem-context", telemCtx],
      ]);
      const log = createLog(parentCtx);

      // First update: not scrolling
      log._updateState({
        path: ["test-log"],
        state: logState.parse({
          region: REGION_500,
          wheelPos: 0,
          scrolling: false,
          empty: true,
          visible: true,
        }),
        type: "log",
        create: () => log,
      });

      // Enter scrollback
      log._updateState({
        path: ["test-log"],
        state: logState.parse({
          region: REGION_500,
          wheelPos: 100,
          scrolling: true,
          empty: false,
          visible: true,
        }),
        type: "log",
        create: () => log,
      });

      const offsetBeforeEviction = log.scrollState.offset;

      // Simulate eviction
      const onChangeCallback = source.onChange.mock.calls[0][0];
      source.evictedCount = 10;
      source.value.mockReturnValue(entries.slice(10));
      onChangeCallback();

      expect(log.scrollState.offset).toBe(offsetBeforeEviction - 10);
    });
  });

  describe("render", () => {
    it("should return undefined for zero-area region", () => {
      const zeroRegion = box.construct({ x: 0, y: 0 }, { width: 0, height: 0 });
      const { log } = setupWithContext([makeEntry(0)], zeroRegion);
      const result = log.render();
      expect(result).toBeUndefined();
    });

    it("should return a cleanup function when not visible", () => {
      const { log } = setupWithContext([makeEntry(0)], REGION_500, {
        visible: false,
      });
      const result = log.render();
      expect(result).toBeTypeOf("function");
    });

    it("should return a cleanup function when visible with entries", () => {
      const entries = Array.from({ length: 10 }, (_, i) => makeEntry(i));
      const { log } = setupWithContext(entries);
      const result = log.render();
      expect(result).toBeTypeOf("function");
    });

    it("should update visibleStart state during render", () => {
      const entries = Array.from({ length: 50 }, (_, i) => makeEntry(i));
      const { log } = setupWithContext(entries);
      log.render();
      // visibleStart should be set to the start of the visible slice
      const expectedStart = Math.max(0, entries.length - log.visibleLineCount);
      expect(log.state.visibleStart).toBe(expectedStart);
    });

    it("should update computedLineHeight state during render", () => {
      const entries = Array.from({ length: 10 }, (_, i) => makeEntry(i));
      const { log } = setupWithContext(entries);
      log.render();
      expect(log.state.computedLineHeight).toBe(log.lineHeight);
    });
  });

  describe("render with scrollback", () => {
    it("should render the correct slice when scrolled back", () => {
      const entries = Array.from({ length: 100 }, (_, i) => makeEntry(i));
      const source = mockLogSource(entries);
      const renderCtx = mockRenderContext();
      const telemCtx = mockTelemContext(source);
      const parentCtx = new Map<string, unknown>([
        ["pluto-theming-context", THEME],
        ["pluto-render-context", renderCtx],
        ["pluto-telem-context", telemCtx],
      ]);
      const log = createLog(parentCtx);

      log._updateState({
        path: ["test-log"],
        state: logState.parse({
          region: REGION_500,
          wheelPos: 0,
          scrolling: false,
          empty: true,
          visible: true,
        }),
        type: "log",
        create: () => log,
      });

      // Enter scrollback
      log._updateState({
        path: ["test-log"],
        state: logState.parse({
          region: REGION_500,
          wheelPos: 100,
          scrolling: true,
          empty: false,
          visible: true,
        }),
        type: "log",
        create: () => log,
      });

      const result = log.render();
      expect(result).toBeTypeOf("function");
      // visibleStart should reflect the scrolled position
      expect(log.state.visibleStart).toBeLessThan(entries.length);
    });
  });

  describe("channel configs and formatting", () => {
    it("should format entries with channel names when showChannelNames is true", () => {
      const entries = Array.from({ length: 5 }, (_, i) => makeEntry(i, 1));
      const { log } = setupWithContext(entries, REGION_500, {
        showChannelNames: true,
        channels: [{ channel: 1 }],
        channelNames: { "1": "Sensor1" },
        selectionStart: 0,
        selectionEnd: 0,
      });
      log.render();
      // selectedText should contain the channel name
      expect(log.state.selectedText).toContain("Sensor1");
    });

    it("should format entries without channel names when showChannelNames is false", () => {
      const entries = Array.from({ length: 5 }, (_, i) => makeEntry(i, 1));
      const { log } = setupWithContext(entries, REGION_500, {
        showChannelNames: false,
        selectionStart: 0,
        selectionEnd: 0,
      });
      log.render();
      // selectedText should NOT contain the channel name
      expect(log.state.selectedText).not.toContain("Sensor1");
    });

    it("should apply precision formatting when channelConfig has precision", () => {
      const entries: LogEntry[] = [
        {
          channelKey: 1,
          timestamp: BigInt(TimeStamp.milliseconds(1000).valueOf()),
          value: "3.14159265",
        },
      ];
      const { log } = setupWithContext(entries, REGION_500, {
        channels: [{ channel: 1, precision: 2 }],
        selectionStart: 0,
        selectionEnd: 0,
      });
      log.render();
      expect(log.state.selectedText).toContain("3.14");
    });

    it("should apply scientific notation when configured", () => {
      const entries: LogEntry[] = [
        {
          channelKey: 1,
          timestamp: BigInt(TimeStamp.milliseconds(1000).valueOf()),
          value: "12345",
        },
      ];
      const { log } = setupWithContext(entries, REGION_500, {
        channels: [{ channel: 1, notation: "scientific", precision: 2 }],
        selectionStart: 0,
        selectionEnd: 0,
      });
      log.render();
      // Should contain scientific notation (uses unicode ᴇ)
      expect(log.state.selectedText).toMatch(/[eEᴇ]/);
    });
  });

  describe("selectedText and selectedLines", () => {
    it("should clear selectedText when selection is negative", () => {
      const entries = Array.from({ length: 5 }, (_, i) => makeEntry(i));
      const { log } = setupWithContext(entries, REGION_500, {
        selectionStart: -1,
        selectionEnd: -1,
      });
      log.render();
      expect(log.state.selectedText).toBe("");
      expect(log.state.selectedLines).toEqual([]);
    });

    it("should set selectedText for a single selected entry", () => {
      const entries = Array.from({ length: 10 }, (_, i) => makeEntry(i));
      const { log } = setupWithContext(entries, REGION_500, {
        selectionStart: 0,
        selectionEnd: 0,
      });
      log.render();
      expect(log.state.selectedText.length).toBeGreaterThan(0);
      expect(log.state.selectedLines).toHaveLength(1);
    });

    it("should set selectedText for a range of selected entries", () => {
      const entries = Array.from({ length: 10 }, (_, i) => makeEntry(i));
      const { log } = setupWithContext(entries, REGION_500, {
        selectionStart: 2,
        selectionEnd: 5,
      });
      log.render();
      // selectedLines should contain entries 2 through 5 inclusive
      expect(log.state.selectedLines).toHaveLength(4);
      expect(log.state.selectedText).toContain("\n");
    });

    it("should handle reversed selection (end < start)", () => {
      const entries = Array.from({ length: 10 }, (_, i) => makeEntry(i));
      const { log } = setupWithContext(entries, REGION_500, {
        selectionStart: 5,
        selectionEnd: 2,
      });
      log.render();
      // Should still produce correct selected text (min to max)
      expect(log.state.selectedLines).toHaveLength(4);
    });

    it("should include color in selectedLines when channel has custom color", () => {
      const entries = Array.from({ length: 5 }, (_, i) => makeEntry(i));
      const { log } = setupWithContext(entries, REGION_500, {
        channels: [{ channel: 1, color: "#ff0000" }],
        selectionStart: 0,
        selectionEnd: 0,
      });
      log.render();
      expect(log.state.selectedLines[0].color).not.toBe("");
    });

    it("should have empty color in selectedLines when no custom color", () => {
      const entries = Array.from({ length: 5 }, (_, i) => makeEntry(i));
      const { log } = setupWithContext(entries, REGION_500, {
        selectionStart: 0,
        selectionEnd: 0,
      });
      log.render();
      expect(log.state.selectedLines[0].color).toBe("");
    });
  });

  describe("render with selection highlighting", () => {
    it("should not render selection when no selection is active", () => {
      const entries = Array.from({ length: 10 }, (_, i) => makeEntry(i));
      const { log } = setupWithContext(entries, REGION_500, {
        selectionStart: -1,
        selectionEnd: -1,
      });
      const result = log.render();
      expect(result).toBeTypeOf("function");
    });

    it("should render selection highlight when selection is within visible range", () => {
      const entries = Array.from({ length: 10 }, (_, i) => makeEntry(i));
      const { log } = setupWithContext(entries, REGION_500, {
        selectionStart: 2,
        selectionEnd: 5,
      });
      const result = log.render();
      expect(result).toBeTypeOf("function");
    });

    it("should render with copyFlash color when copyFlash is true", () => {
      const entries = Array.from({ length: 10 }, (_, i) => makeEntry(i));
      const { log } = setupWithContext(entries, REGION_500, {
        selectionStart: 0,
        selectionEnd: 2,
        copyFlash: true,
      });
      const result = log.render();
      expect(result).toBeTypeOf("function");
    });
  });

  describe("render scrollbar", () => {
    it("should render scrollbar when scrolling with many entries", () => {
      const entries = Array.from({ length: 200 }, (_, i) => makeEntry(i));
      const source = mockLogSource(entries);
      const renderCtx = mockRenderContext();
      const telemCtx = mockTelemContext(source);
      const parentCtx = new Map<string, unknown>([
        ["pluto-theming-context", THEME],
        ["pluto-render-context", renderCtx],
        ["pluto-telem-context", telemCtx],
      ]);
      const log = createLog(parentCtx);

      log._updateState({
        path: ["test-log"],
        state: logState.parse({
          region: REGION_500,
          wheelPos: 0,
          scrolling: false,
          empty: true,
          visible: true,
        }),
        type: "log",
        create: () => log,
      });

      log._updateState({
        path: ["test-log"],
        state: logState.parse({
          region: REGION_500,
          wheelPos: 100,
          scrolling: true,
          empty: false,
          visible: true,
        }),
        type: "log",
        create: () => log,
      });

      const result = log.render();
      expect(result).toBeTypeOf("function");
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
      const { log } = setupWithContext(entries, REGION_500, {
        showChannelNames: true,
        channels: [{ channel: 1 }, { channel: 2 }, { channel: 3 }],
        channelNames: { "1": "Temperature", "2": "Pressure", "3": "Humidity" },
        selectionStart: 0,
        selectionEnd: 3,
      });
      log.render();
      expect(log.state.selectedLines).toHaveLength(4);
      expect(log.state.selectedText).toContain("Temperature");
      expect(log.state.selectedText).toContain("Pressure");
      expect(log.state.selectedText).toContain("Humidity");
    });

    it("should apply per-channel colors from channelConfigs", () => {
      const entries = [makeEntry(0, 1), makeEntry(1, 2)];
      const { log } = setupWithContext(entries, REGION_500, {
        channels: [
          { channel: 1, color: "#ff0000" },
          { channel: 2, color: "#00ff00" },
        ],
        selectionStart: 0,
        selectionEnd: 1,
      });
      log.render();
      expect(log.state.selectedLines[0].color).not.toBe("");
      expect(log.state.selectedLines[1].color).not.toBe("");
      expect(log.state.selectedLines[0].color).not.toBe(
        log.state.selectedLines[1].color,
      );
    });
  });

  describe("theme handling", () => {
    it("should work with light theme", () => {
      const lightTheme = themeZ.parse(SYNNAX_LIGHT);
      const entries = Array.from({ length: 5 }, (_, i) => makeEntry(i));
      const { log } = setupWithContext(entries, REGION_500, {}, lightTheme);
      expect(log.lineHeight).toBeGreaterThan(0);
      expect(log.entries).toHaveLength(5);
    });
  });

  describe("afterDelete", () => {
    it("should clean up telem and erase render region", () => {
      const entries = Array.from({ length: 5 }, (_, i) => makeEntry(i));
      const { log, source, renderCtx } = setupWithContext(entries);
      log.afterDelete();
      expect(source.cleanup).toHaveBeenCalled();
      expect(renderCtx.erase).toHaveBeenCalled();
    });
  });
});
