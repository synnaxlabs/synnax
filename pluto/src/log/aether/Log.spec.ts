// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { box, color, TimeStamp, xy } from "@synnaxlabs/x";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { type LogEntry } from "@/telem/aether/telem";
import { Log, logState } from "@/log/aether/Log";
import { type Theme, SYNNAX_DARK } from "@/theming/base/theme";
import { themeZ } from "@/theming/base/theme";

const MockSender = { send: vi.fn() };

const THEME: Theme = themeZ.parse(SYNNAX_DARK);

const mockLogSource = (entries: LogEntry[] = []) => ({
  value: vi.fn(() => entries),
  evictedCount: 0,
  cleanup: vi.fn(),
  onChange: vi.fn(() => () => {}),
});

const mockRenderContext = () => ({
  loop: { set: vi.fn() },
  erase: vi.fn(),
  scissor: vi.fn(() => vi.fn()),
  lower2d: {
    canvas: { width: 800, height: 600 },
    getContext: vi.fn(),
  },
});

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

const makeEntry = (
  i: number,
  channelKey: number = 1,
  channelName: string = "ch1",
): LogEntry => ({
  channelKey,
  channelName,
  channelPadding: "",
  timestamp: BigInt(TimeStamp.milliseconds(i * 1000).valueOf()),
  value: String(i),
});

const setupWithContext = (
  entries: LogEntry[] = [],
  region: box.Box = REGION_500,
): { log: Log; source: ReturnType<typeof mockLogSource> } => {
  const source = mockLogSource(entries);
  const renderCtx = mockRenderContext();
  const telemCtx = mockTelemContext(source);
  const parentCtx = new Map<string, unknown>([
    ["pluto-theming-context", THEME],
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
  });
  log._updateState({
    path: ["test-log"],
    state,
    type: "log",
    create: () => log,
  });
  return { log, source };
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
});
