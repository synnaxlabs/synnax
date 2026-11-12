// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, log, type status } from "@synnaxlabs/client";
import { type theme } from "@synnaxlabs/lyra/theme";
import {
  bounds,
  box,
  color,
  compare,
  DataType,
  type destructor,
  notation,
  text,
  TimeStamp,
  type TimeZone,
  xy,
} from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { noopLogSourceSpec } from "@/log/aether/telem/noop";
import {
  type LogEntry,
  type LogSource,
  logSourceSpecZ,
} from "@/log/aether/telem/types";
import { telem } from "@/telem/aether";
import { theming } from "@/theming/aether";
import { Draw2D } from "@/vis/draw2d";
import { render } from "@/vis/render";

export const logStateZ = log.logZ
  .pick({
    channels: true,
    timestampPrecision: true,
    channelNamesHidden: true,
    timestampsHidden: true,
    timeZone: true,
  })
  .extend({
    region: box.box,
    wheelPos: z.number(),
    scrolling: z.boolean(),
    resumedAt: z.number().default(0),
    empty: z.boolean(),
    loading: z.boolean().default(false),
    blank: z.boolean().default(false),
    visible: z.boolean(),
    // channelNames: server-side display names (fetched fresh, never persisted)
    // channelDataTypes: server-side data types (fetched fresh, never persisted)
    channelNames: z.record(z.string(), z.string()).default({}),
    channelDataTypes: z.record(z.string(), z.string()).default({}),
    hiddenChannels: z.array(channel.keyZ).default([]),
    telem: logSourceSpecZ.default(noopLogSourceSpec),
    font: text.levelZ.default("p"),
    color: color.colorZ.default(color.ZERO),
    overshoot: xy.xyZ.default({ x: 0, y: 0 }),
    // position is the time of the entry at the bottom of a paused view: a decimal count
    // of nanoseconds, since a number cannot hold it exactly. Empty while live.
    position: z.string().default(""),
    selectionStart: z.number().default(-1),
    selectionEnd: z.number().default(-1),
    visibleStart: z.number().default(0),
    computedLineHeight: z.number().default(0),
    // The entry index of each row on screen, top to bottom.
    rowEntries: z.array(z.number()).default([]),
  });

const lineZ = z.object({ text: z.string(), color: z.string() });

/** A selected line as a copy writes it. An empty color follows the theme. */
export interface Line extends z.infer<typeof lineZ> {}

const selectionZ = z.object({ lines: z.array(lineZ), partial: z.boolean() });

/**
 * The selected lines as a copy writes them. partial is true when the selection is a
 * select-all and the channels store values the log does not hold.
 */
export interface Selection extends z.infer<typeof selectionZ> {}

export const methodsZ = {
  /** Returns the selection at the time of the call. */
  selection: z.function({ input: z.tuple([]), output: selectionZ }),
};

/**
 * A selection end that follows the newest entry, so select-all needs no entry count
 * from the worker.
 */
export const SELECT_ALL_END = Number.MAX_SAFE_INTEGER;

const SCROLLBAR_RENDER_THRESHOLD = 0.98;
const SCROLLBAR_WIDTH = 6;
// Floor on the thumb so it stays visible at the entry cap, where the proportional
// height falls below a pixel.
const SCROLLBAR_MIN_HEIGHT = 32;
const CANVAS: render.Canvas2DVariant = "lower2d";
// A paused view this many lines or fewer from either end of the entries reads more.
const LOAD_MARGIN = 200;
const CONTENT_PADDING = 6;
// The fewest characters of a value beside its prefix. With less room, the value starts
// on the row below its prefix and uses the full width.
const MIN_VALUE_COLUMNS = 16;

// Per-theme prefix color muting — multiplied with the base color's HSLA.
const DARK_PREFIX = { hue: 1, saturation: 0.8, lightness: 0.85, alpha: 0.95 };
const LIGHT_PREFIX = { hue: 1, saturation: 0.8, lightness: 0.8, alpha: 0.8 };

interface InternalState {
  theme: theme.Theme;
  render: render.Context;
  draw2d: Draw2D;
  telem: telem.MemoizedSource<LogEntry[], LogSource>;
  configs: Record<string, log.ChannelEntry>;
  textColor: color.Color;
  prefixColors: Record<string, color.Color>;
  defaultPrefixColor: color.Color;
  // Cached per-channel value colors to avoid repeated lookups in render loop.
  valueColors: Record<string, color.Color>;
  // Display name per channel (alias if set, otherwise server name).
  displayNames: Record<string, string>;
  // Padding to align channel names to the same width in the log.
  namePadding: Record<string, string>;
  charWidth: number;
  lineHeight: number;
  // Downward nudge aligning the selection band with glyph ink (see
  // Draw2D.measureInkOffsetY).
  selectionOffsetY: number;
  tsLen: number;
  selectionColor: color.Color;
  stopListeningTelem?: destructor.Destructor;
  stopListeningRestore?: destructor.Destructor;
}

// One line of pixels on screen. A wrapped entry has one row per piece of its value. A
// day row shows its text in prefix, above entry.
interface Row {
  entry: number;
  prefix: string;
  value: string;
  channelKey: string;
  first: boolean;
  day?: boolean;
}

// How an entry splits into rows. The first lead rows hold no value.
interface EntryLayout {
  entry: number;
  prefix: string;
  value: string;
  channelKey: string;
  indent: string;
  columns: number;
  lead: number;
  rows: number;
  // starts holds where each row of a value with wide characters starts.
  starts?: number[];
}

interface ScrollbackState {
  offset: number;
  // tail is the rows of the entry at offset that sit below the view.
  tail: number;
  scrollRef: number;
  // moved is the rows the wheel has already moved the view.
  moved: number;
  // Whether the viewport has moved off the newest entry since the pause began.
  awayFromEnd: boolean;
}

// The entries at the selection ends and at the bottom of the paused view.
interface Anchors {
  start?: LogEntry;
  end?: LogEntry;
  scroll?: LogEntry;
}

const ZERO_SCROLLBACK: ScrollbackState = {
  offset: 0,
  tail: 0,
  scrollRef: 0,
  moved: 0,
  awayFromEnd: false,
};

const muteColor = (c: color.Crude, theme: theme.Theme): color.Color => {
  const p = theme.key === "synnaxDark" ? DARK_PREFIX : LIGHT_PREFIX;
  const [h, s, l, a] = color.hsla(c);
  return color.fromHSLA([
    Math.min(360, Math.round(h * p.hue)),
    Math.min(100, Math.round(s * p.saturation)),
    Math.min(100, Math.round(l * p.lightness)),
    Math.min(1, a * p.alpha),
  ]);
};

const dayText = ({ timestamp }: LogEntry, timeZone: TimeZone): string =>
  new Date(new TimeStamp(timestamp).milliseconds).toLocaleDateString("en-US", {
    weekday: "short",
    month: "short",
    day: "numeric",
    year: "numeric",
    timeZone: timeZone === "UTC" ? "UTC" : undefined,
  });

const sameDay = (a: LogEntry, b: LogEntry, timeZone: TimeZone): boolean =>
  new TimeStamp(a.timestamp).isSameDay(new TimeStamp(b.timestamp), timeZone);

// East Asian characters take two columns of a monospace font. An emoji takes two
// already, as a surrogate pair.
const WIDE =
  /[\u1100-\u115f\u2e80-\ua4cf\uac00-\ud7a3\uf900-\ufaff\ufe30-\ufe6f\uff00-\uff60\uffe0-\uffe6]/;

// Returns where each row of a value with wide characters starts.
const wrapStarts = (value: string, columns: number): number[] => {
  const starts = [0];
  let used = 0;
  for (let i = 0; i < value.length; i++) {
    const code = value.charCodeAt(i);
    const width = WIDE.test(value[i]) ? 2 : 1;
    if (used > 0 && used + width > columns && (code < 0xdc00 || code > 0xdfff)) {
      starts.push(i);
      used = 0;
    }
    used += width;
  }
  return starts;
};

const INTEGER = /^-?\d+$/;

// Returns the sample in the number format of its channel. A sample shows as it arrives
// when the channel holds no numbers, its data type has not loaded, or a calculated
// channel changed type after the sample arrived.
const formatNumber = (
  value: string,
  dataType: string | undefined,
  { precision, notation: valueNotation }: log.ChannelEntry,
): string => {
  if (dataType == null) return value;
  const type = new DataType(dataType);
  if (type.isInteger && INTEGER.test(value))
    return notation.stringifyNumber(BigInt(value), precision, valueNotation);
  if (!type.isFloat || value.trim() === "") return value;
  const number = Number(value);
  if (Number.isNaN(number)) return value;
  return notation.stringifyNumber(number, precision, valueNotation);
};

const sameEntries = (a: number[], b: number[]): boolean =>
  a.length === b.length && a.every((entry, i) => entry === b[i]);

// Moves a wrap point off the second half of a surrogate pair, so a row never splits an
// emoji.
const wrapPoint = (value: string, at: number): number => {
  const code = value.charCodeAt(at);
  return code >= 0xdc00 && code <= 0xdfff ? at + 1 : at;
};

export class Log
  extends aether.Leaf<typeof logStateZ, InternalState, typeof methodsZ>
  implements aether.HandlersFromSchema<typeof methodsZ>
{
  static readonly TYPE = "log";
  static readonly z = logStateZ;
  schema = Log.z;
  methods = methodsZ;
  entries: LogEntry[] = [];
  // Copied, not shared: the scrollback branch mutates this in place.
  scrollState: ScrollbackState = { ...ZERO_SCROLLBACK };
  // Render key for requestRender — allocated once, not per call.
  private renderKey: string = "";
  // A source change can insert or evict entries before the anchors, so their positions
  // are found again by identity.
  private anchors: Anchors = {};
  // The position to return to, until the entries reach it.
  private seeking: bigint | null = null;
  private readonly wideRows = new WeakMap<
    LogEntry,
    { columns: number; starts: number[] }
  >();

  get lineHeight(): number {
    return this.internal.lineHeight;
  }

  get totalHeight(): number {
    return Math.ceil(this.entries.length * this.internal.lineHeight);
  }

  get visibleLineCount(): number {
    return this.calcVisibleLineCount(this.internal.lineHeight);
  }

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    // Read before any setState below, which overwrites prevState.
    const wasVisible = this.prevState.visible;
    i.render = render.Context.use(ctx);
    i.stopListeningRestore?.();
    i.stopListeningRestore = i.render.on2DRestore(() => {
      if (this.state.visible) this.requestRender();
    });
    i.theme = theming.use(ctx);
    i.draw2d = new Draw2D(i.render[CANVAS], i.theme);
    if (this.renderKey === "") {
      this.renderKey = `${this.type}-${this.key}`;
      if (this.state.scrolling && this.state.position !== "")
        this.seeking = BigInt(this.state.position);
    }

    if (color.isZero(this.state.color)) i.textColor = i.theme.colors.gray.l11;
    else i.textColor = this.state.color;

    i.lineHeight = i.theme.typography[this.state.font].size * i.theme.sizes.base;
    i.charWidth = i.draw2d.measureCharWidth(this.state.font);
    i.selectionOffsetY = i.draw2d.measureInkOffsetY(this.state.font, i.lineHeight);
    i.tsLen =
      this.state.timestampPrecision === 0 ? 8 : 9 + this.state.timestampPrecision;

    const channelEntries = this.state.channels;
    const prevEntries = this.prevState.channels;
    const channelsChanged = channelEntries !== prevEntries;
    const channelKeys = channelEntries.map((e) => e.channel);
    const configs: Record<string, log.ChannelEntry> = {};
    for (const entry of channelEntries) configs[entry.channel] = entry;

    // Rebuild color caches only when configs or base color changed.
    const colorChanged = !color.equals(this.state.color, this.prevState.color);
    if (channelsChanged || colorChanged || i.prefixColors == null) {
      i.defaultPrefixColor = muteColor(i.textColor, i.theme);
      i.prefixColors = {};
      i.valueColors = {};
      for (const [key, cfg] of Object.entries(configs))
        if (!color.isZero(cfg.color)) {
          i.prefixColors[key] = muteColor(cfg.color, i.theme);
          i.valueColors[key] = color.construct(cfg.color);
        }
    }
    i.configs = configs;

    i.selectionColor = color.setAlpha(i.theme.colors.primary.z, 0.25);

    i.telem = telem.useSource(ctx, this.state.telem, i.telem);

    // Always call setChannels — the source short-circuits if unchanged. This handles
    // both initial setup (where prevState === state) and subsequent changes.
    i.telem.wrapped.setChannels?.(channelKeys);
    const { hiddenChannels } = this.state;
    // A new filter reloads the entries, so a paused view returns to its time.
    if (
      this.state.scrolling &&
      this.seeking == null &&
      compare.primitiveArrays(hiddenChannels, this.prevState.hiddenChannels) !==
        compare.EQUAL
    )
      this.seeking =
        this.entries[this.scrollState.offset - 1]?.timestamp.valueOf() ?? null;
    i.telem.wrapped.setFilter?.({ hiddenChannels });
    const { data } = i.telem.value();
    this.entries = data;

    // Compute display names (alias takes priority over server name) and padding.
    const namesChanged =
      channelsChanged ||
      this.state.channelNames !== this.prevState.channelNames ||
      i.displayNames == null;
    if (namesChanged) {
      i.displayNames = {};
      for (const [key, cfg] of Object.entries(configs))
        i.displayNames[key] = cfg.alias || this.state.channelNames[key] || String(key);
      const maxLen = Math.max(0, ...Object.values(i.displayNames).map((n) => n.length));
      i.namePadding = {};
      for (const [key, name] of Object.entries(i.displayNames))
        i.namePadding[key] = " ".repeat(maxLen - name.length);
    }

    const { scrolling, wheelPos } = this.state;
    const lh = i.lineHeight;

    // Scrollback: snapshot the current position when the user first pauses scrolling.
    const justEnteredScrollback = this.state.scrolling && !this.prevState.scrolling;
    if (justEnteredScrollback) {
      this.seeking = null;
      const off = this.entries.length;
      this.scrollState = {
        offset: off,
        tail: 0,
        scrollRef: this.state.wheelPos,
        moved: 0,
        awayFromEnd: false,
      };
    } else if (scrolling) {
      // Adjust viewport offset based on wheel delta from the snapshot reference.
      const { scrollState } = this;
      const dist = Math.ceil((wheelPos - scrollState.scrollRef) / lh);
      if (dist !== scrollState.moved) this.seeking = null;
      this.moveView(dist - scrollState.moved, lh);
      scrollState.moved = dist;
      // Clamp: if scrolled to top, lock wheel position to prevent over-scroll.
      if (this.atTop(this.calcTop(lh))) {
        const { wheelPos: locked } = this.prevState;
        scrollState.moved = Math.ceil((locked - scrollState.scrollRef) / lh);
        this.setState((s) => ({ ...s, wheelPos: locked }));
      }
      // Auto-resume live mode when the viewport is back on the newest entry. A pause
      // taken at the bottom starts there, so it takes a return or a downward scroll. A
      // view that waits for its time stays paused.
      if (scrollState.offset < this.entries.length || scrollState.tail > 0)
        scrollState.awayFromEnd = true;
      else if (
        this.seeking == null &&
        i.telem.wrapped.live !== false &&
        (scrollState.awayFromEnd || dist < 0)
      ) {
        this.setState((s) => ({
          ...s,
          scrolling: false,
          resumedAt: performance.now(),
        }));
        i.telem.wrapped.resume?.();
      }
    }

    this.setAnchors(this.state.selectionStart, this.state.selectionEnd);
    this.syncStatus(i.telem.value().status);
    i.stopListeningTelem?.();
    i.stopListeningTelem = i.telem.onChange(() => {
      const { data, status } = this.internal.telem.value();
      this.entries = data;
      this.reanchor();
      this.syncStatus(status);
      this.loadNearEdges();
      if (this.state.visible) this.requestRender();
    });
    if (!scrolling && this.prevState.scrolling) i.telem.wrapped.resume?.();
    this.loadNearEdges();
    if (!this.state.visible && !wasVisible) return;
    this.requestRender();
  }

  private loadNearEdges(): void {
    if (!this.state.scrolling) return;
    const { wrapped } = this.internal.telem;
    const { offset } = this.scrollState;
    if (offset - this.visibleLineCount <= LOAD_MARGIN) wrapped.loadOlder?.();
    if (this.entries.length - offset <= LOAD_MARGIN) wrapped.loadNewer?.();
  }

  // A loading log is not empty, since its stored entries have not arrived yet.
  private syncStatus(s: status.Status): void {
    const loading = s.variant === "loading";
    const empty = this.entries.length === 0 && !loading;
    const blank = this.entries.length === 0 && loading;
    if (
      empty === this.state.empty &&
      loading === this.state.loading &&
      blank === this.state.blank
    )
      return;
    this.setState((p) => ({ ...p, empty, loading, blank }));
  }

  afterDelete(): void {
    const { telem, stopListeningRestore } = this.internal;
    stopListeningRestore?.();
    telem.cleanup?.();
    this.requestRender();
  }

  private requestRender(): void {
    this.internal.render.loop.set({
      key: this.renderKey,
      render: () => this.render(),
      priority: "high",
      canvases: [CANVAS],
    });
  }

  private calcRowCapacity(lh: number): number {
    return Math.floor((box.height(this.state.region) - CONTENT_PADDING * 2) / lh);
  }

  private calcVisibleLineCount(lh: number): number {
    return Math.min(this.calcRowCapacity(lh), this.entries.length);
  }

  // Returns the view that starts at the first row of the oldest entry.
  private calcTop(lh: number): { offset: number; tail: number } {
    const capacity = this.calcRowCapacity(lh);
    let rows = 0;
    for (let end = 1; end <= this.entries.length; end++) {
      const { rows: entryRows } = this.layoutEntry(end - 1);
      const total = rows + entryRows + (this.startsDay(end - 1) ? 1 : 0);
      if (total > capacity) {
        if (total - capacity >= entryRows)
          return { offset: Math.max(1, end - 1), tail: 0 };
        return { offset: end, tail: total - capacity };
      }
      rows = total;
    }
    return { offset: this.entries.length, tail: 0 };
  }

  private position(offset: number, tail: number): number {
    if (tail === 0) return offset;
    return offset - tail / this.layoutEntry(offset - 1).rows;
  }

  private calcMinOffset(lh: number): number {
    return this.calcTop(lh).offset;
  }

  private atTop(top: { offset: number; tail: number }): boolean {
    const { offset, tail } = this.scrollState;
    return offset < top.offset || (offset === top.offset && tail >= top.tail);
  }

  // Moves the paused view by rows, up when rows is positive, and stops at either end.
  private moveView(rows: number, lh: number): void {
    const view = this.scrollState;
    if (view.offset > this.entries.length)
      [view.offset, view.tail] = [this.entries.length, 0];
    if (view.offset > 0)
      view.tail = Math.min(view.tail, this.layoutEntry(view.offset - 1).rows - 1);
    const top = this.calcTop(lh);
    if (view.offset < top.offset) [view.offset, view.tail] = [top.offset, top.tail];
    for (; rows > 0 && !this.atTop(top); rows--)
      if (view.tail < this.layoutEntry(view.offset - 1).rows - 1) view.tail++;
      else [view.offset, view.tail] = [view.offset - 1, 0];
    for (; rows < 0 && (view.offset < this.entries.length || view.tail > 0); rows++)
      if (view.tail > 0) view.tail--;
      else {
        view.offset++;
        view.tail = this.layoutEntry(view.offset - 1).rows - 1;
      }
  }

  private layoutEntry(entry: number): EntryLayout {
    const { prefix, value, channelKey } = this.formatEntry(this.entries[entry]);
    const width = box.width(this.state.region) - CONTENT_PADDING * 2 - SCROLLBAR_WIDTH;
    const fullColumns = Math.floor(width / this.internal.charWidth);
    const columns = fullColumns - prefix.length;
    const base = { entry, prefix, value, channelKey };
    const wide = WIDE.test(value);
    const startsOf = (cols: number): number[] => this.wrapStarts(entry, value, cols);
    const rowsOf = (cols: number): { rows: number; starts?: number[] } => {
      if (value.length === 0) return { rows: 1 };
      if (!wide) return { rows: Math.ceil(value.length / cols) };
      const starts = startsOf(cols);
      return { rows: starts.length, starts };
    };
    const fits =
      value.length === 0 ||
      (wide ? startsOf(columns).length === 1 : value.length <= columns);
    if (fits || columns >= MIN_VALUE_COLUMNS)
      return {
        ...base,
        indent: " ".repeat(prefix.length),
        columns,
        lead: 0,
        ...rowsOf(columns),
      };
    const ownColumns = Math.max(1, fullColumns);
    const own = rowsOf(ownColumns);
    return {
      ...base,
      indent: "",
      columns: ownColumns,
      lead: 1,
      ...own,
      rows: 1 + own.rows,
    };
  }

  // Returns the row starts of a wide value, which only change with the width.
  private wrapStarts(entry: number, value: string, columns: number): number[] {
    const key = this.entries[entry];
    const held = this.wideRows.get(key);
    if (held?.columns === columns) return held.starts;
    const starts = wrapStarts(value, columns);
    this.wideRows.set(key, { columns, starts });
    return starts;
  }

  private rowOf(layout: EntryLayout, row: number): Row {
    const { entry, prefix, value, channelKey, indent, columns, lead, starts } = layout;
    const start = (row - lead) * columns;
    const text =
      starts != null
        ? value.slice(starts[row - lead], starts[row - lead + 1])
        : value.slice(wrapPoint(value, start), wrapPoint(value, start + columns));
    return {
      entry,
      prefix: row === 0 ? prefix : indent,
      value: row < lead ? "" : text,
      channelKey,
      first: row === 0,
    };
  }

  // Returns whether a day row sits above the entry. Times with no date need no day.
  private startsDay(entry: number): boolean {
    if (this.state.timestampsHidden) return false;
    return (
      entry === 0 ||
      !sameDay(this.entries[entry], this.entries[entry - 1], this.state.timeZone)
    );
  }

  private dayRow(entry: number): Row {
    const prefix = dayText(this.entries[entry], this.state.timeZone);
    return { entry, prefix, value: "", channelKey: "", first: false, day: true };
  }

  // Lays out the rows that fit above the entry at end, less its bottom tail rows, and
  // cuts the top entry to fit.
  private fitRows(end: number, tail: number, capacity: number): Row[] {
    const rows: Row[] = [];
    let last = tail;
    for (let i = end - 1; i >= 0 && rows.length < capacity; i--) {
      const layout = this.layoutEntry(i);
      const bottom = layout.rows - 1 - Math.min(last, layout.rows - 1);
      const top = Math.max(0, bottom + 1 - (capacity - rows.length));
      for (let row = bottom; row >= top; row--) rows.push(this.rowOf(layout, row));
      if (rows.length < capacity && this.startsDay(i)) rows.push(this.dayRow(i));
      last = 0;
    }
    return rows.reverse();
  }

  // The top row always names the day of the rows below it.
  private layoutRows(end: number, tail: number, lh: number): Row[] {
    const capacity = this.calcRowCapacity(lh);
    const rows = this.fitRows(end, tail, capacity);
    if (this.state.timestampsHidden || capacity < 2 || rows.length === 0) return rows;
    if (rows[0].day) return rows;
    const pinned = this.fitRows(end, tail, capacity - 1);
    if (pinned[0].day) return pinned;
    return [this.dayRow(pinned[0].entry), ...pinned];
  }

  render(): render.Cleanup | undefined {
    const { render: renderCtx, lineHeight: lh, draw2d } = this.internal;
    const region = this.state.region;
    if (this.deleted || box.areaIsZero(region)) return undefined;
    if (!this.state.visible) return () => renderCtx.erase(region, xy.ZERO, CANVAS);

    const end = this.state.scrolling ? this.scrollState.offset : this.entries.length;
    const tail = this.state.scrolling ? this.scrollState.tail : 0;
    const rows = this.layoutRows(end, tail, lh);
    const sliceStart = rows.length > 0 ? rows[0].entry : end;
    const rowEntries = rows.map((row) => row.entry);
    const shown = this.seeking ?? this.entries[end - 1]?.timestamp.valueOf() ?? "";
    const position = this.state.scrolling ? shown.toString() : "";

    if (
      this.state.position !== position ||
      this.state.visibleStart !== sliceStart ||
      this.state.computedLineHeight !== lh ||
      !sameEntries(this.state.rowEntries, rowEntries)
    )
      this.setState((s) => ({
        ...s,
        position,
        visibleStart: sliceStart,
        computedLineHeight: lh,
        rowEntries,
      }));

    const reg = this.state.region;
    const clearScissor = renderCtx.scissor(reg, xy.ZERO, [CANVAS]);
    this.renderSelection(draw2d, rowEntries, lh);
    this.renderElements(draw2d, rows, lh);
    this.renderScrollbar(draw2d, rows);
    clearScissor();
    const eraseRegion = box.copy(this.state.region);
    return ({ canvases }) =>
      renderCtx.erase(eraseRegion, this.state.overshoot, ...canvases);
  }

  // Returns where the rows sit in the times the source stores, from 0 to 1, or null
  // when the source does not know its oldest time.
  private timeShare(rows: Row[]): { top: number; bottom: number } | null {
    const range = this.internal.telem.wrapped.timeRange;
    if (range == null || rows.length === 0) return null;
    const start = range.start.valueOf();
    const total = Number(range.end.valueOf() - start);
    if (total <= 0) return null;
    const share = (row: Row): number =>
      Number(this.entries[row.entry].timestamp.valueOf() - start) / total;
    const clamp = (value: number): number =>
      bounds.clamp({ lower: 0, upper: 1 }, value);
    return { top: clamp(share(rows[0])), bottom: clamp(share(rows[rows.length - 1])) };
  }

  private renderScrollbar(draw2d: Draw2D, rows: Row[]): void {
    if (!this.state.scrolling) return;
    const reg = this.state.region;
    const regHeight = box.height(reg);
    const shown = this.timeShare(rows);
    // A view that spans the stored times can still hide lines, which only line counts
    // can place.
    if (shown != null && shown.bottom - shown.top < SCROLLBAR_RENDER_THRESHOLD) {
      const proportional = (shown.bottom - shown.top) * regHeight;
      const height = Math.min(Math.max(proportional, SCROLLBAR_MIN_HEIGHT), regHeight);
      const outside = shown.top + 1 - shown.bottom;
      const progress = outside <= 0 ? 0 : shown.top / outside;
      this.drawThumb(draw2d, box.top(reg) + progress * (regHeight - height), height);
      return;
    }
    const { totalHeight } = this;
    if (totalHeight <= 0) return;
    const top = this.calcTop(this.internal.lineHeight);
    // Wrapped rows fit fewer entries on screen, so they shrink the thumb.
    const proportionalHeight =
      (regHeight / totalHeight) * regHeight * (top.offset / this.visibleLineCount);
    const upper = regHeight * SCROLLBAR_RENDER_THRESHOLD;
    if (proportionalHeight >= upper && top.tail === 0) return;
    const height = bounds.clamp(
      { lower: SCROLLBAR_MIN_HEIGHT, upper },
      proportionalHeight,
    );
    // The thumb travels the track less its own height, over the entries that sit
    // outside the viewport. Rows below the view count as a part of their entry.
    const { offset, tail } = this.scrollState;
    const scrollRange = this.entries.length - this.position(top.offset, top.tail);
    const distFromEnd = this.entries.length - this.position(offset, tail);
    const progress =
      scrollRange <= 0
        ? 0
        : bounds.clamp({ lower: 0, upper: 1 }, distFromEnd / scrollRange);
    this.drawThumb(
      draw2d,
      box.bottom(reg) - height - progress * (regHeight - height),
      height,
    );
  }

  private drawThumb(draw2d: Draw2D, y: number, height: number): void {
    const reg = this.state.region;
    draw2d.container({
      region: box.construct(
        { x: box.right(reg) - SCROLLBAR_WIDTH, y },
        { width: SCROLLBAR_WIDTH, height },
      ),
      bordered: false,
      backgroundColor: (t: theme.Theme) => t.colors.gray.l6,
    });
  }

  // A selection endpoint past the last entry anchors to the last entry, unless it is
  // SELECT_ALL_END, which stays open. A select-all leaves its start open too, so it
  // covers the entries that older pages add.
  private setAnchors(selectionStart: number, selectionEnd: number): void {
    const last = this.entries.length - 1;
    const anchor = (i: number): LogEntry | undefined =>
      i === SELECT_ALL_END ? undefined : this.entries[Math.min(i, last)];
    const all = selectionStart === 0 && selectionEnd === SELECT_ALL_END;
    this.anchors = {
      start: all ? undefined : anchor(selectionStart),
      end: anchor(selectionEnd),
      scroll: this.state.scrolling
        ? this.entries[this.scrollState.offset - 1]
        : undefined,
    };
  }

  // Returns the position of the entry, or -1 when it was evicted.
  private locate(entry: LogEntry): number {
    const time = entry.timestamp.valueOf();
    let [lo, hi] = [0, this.entries.length];
    while (lo < hi) {
      const mid = (lo + hi) >>> 1;
      if (this.entries[mid].timestamp.valueOf() < time) lo = mid + 1;
      else hi = mid;
    }
    for (let i = lo; i < this.entries.length; i++) {
      if (this.entries[i] === entry) return i;
      if (this.entries[i].timestamp.valueOf() !== time) break;
    }
    return -1;
  }

  // Moves the paused view and the selection back onto their anchored entries. An
  // evicted anchor maps to the oldest entry.
  private reanchor(): void {
    const { start, end, scroll } = this.anchors;
    if (this.seeking != null && this.state.scrolling) this.seek(this.seeking);
    else if (scroll != null && this.state.scrolling) {
      const offset = Math.max(
        this.calcMinOffset(this.internal.lineHeight),
        this.locate(scroll) + 1,
      );
      this.scrollState.offset = offset;
      this.anchors.scroll = this.entries[offset - 1];
    } else if (this.state.scrolling) {
      this.moveView(0, this.internal.lineHeight);
      this.anchors.scroll = this.entries[this.scrollState.offset - 1];
    }
    const { selectionStart, selectionEnd } = this.state;
    if (selectionStart < 0 || selectionEnd < 0 || (start == null && end == null))
      return;
    const newStart = start == null ? selectionStart : this.locate(start);
    const newEnd = end == null ? selectionEnd : this.locate(end);
    if (newStart < 0 && newEnd < 0) {
      this.anchors = { scroll: this.anchors.scroll };
      this.setState((s) => ({ ...s, selectionStart: -1, selectionEnd: -1 }));
      return;
    }
    const [nextStart, nextEnd] = [Math.max(0, newStart), Math.max(0, newEnd)];
    this.setAnchors(nextStart, nextEnd);
    if (nextStart === selectionStart && nextEnd === selectionEnd) return;
    this.setState((s) => ({ ...s, selectionStart: nextStart, selectionEnd: nextEnd }));
  }

  // Waits at the oldest entry while older pages can still reach the time. When no
  // stored sample does, the view stays there. Entries that share the time all show.
  private seek(time: bigint): void {
    const { lineHeight, telem } = this.internal;
    this.moveView(0, lineHeight);
    if (this.entries.length === 0) return;
    const reached = time >= this.entries[0].timestamp.valueOf();
    let lo = 0;
    if (reached) {
      let hi = this.entries.length;
      while (lo < hi) {
        const mid = (lo + hi) >>> 1;
        if (this.entries[mid].timestamp.valueOf() <= time) lo = mid + 1;
        else hi = mid;
      }
    }
    this.scrollState.offset = Math.max(this.calcMinOffset(lineHeight), lo);
    this.scrollState.tail = 0;
    if (!reached && telem.wrapped.exhausted === false) return;
    this.seeking = null;
    // A view that lands on the newest entry starts there, as a pause does.
    this.scrollState.awayFromEnd = false;
    this.anchors.scroll = this.entries[this.scrollState.offset - 1];
  }

  private renderSelection(draw2d: Draw2D, rowEntries: number[], lh: number): void {
    const { selectionStart, selectionEnd } = this.state;
    if (selectionStart < 0 || selectionEnd < 0) return;
    const selMin = Math.min(selectionStart, selectionEnd);
    const selMax = Math.max(selectionStart, selectionEnd);
    const selected = (entry: number) => entry >= selMin && entry <= selMax;
    const highlightStart = rowEntries.findIndex(selected);
    if (highlightStart === -1) return;
    const highlightEnd = rowEntries.findLastIndex(selected);
    const reg = this.state.region;
    const rowCount = highlightEnd - highlightStart + 1;
    draw2d.container({
      region: box.construct(
        xy.translate(box.topLeft(reg), {
          x: 0,
          y: highlightStart * lh + CONTENT_PADDING + this.internal.selectionOffsetY,
        }),
        { width: box.width(reg), height: rowCount * lh },
      ),
      bordered: false,
      rounded: false,
      backgroundColor: this.internal.selectionColor,
    });
  }

  // channelNamesHidden is read from state (O(1)) rather than derived by scanning all
  // entries (O(n)). The render loop below is already O(n) over visible entries — adding
  // a second O(n) scan here just to answer a yes/no question would double the per-frame
  // work at up to 60fps.
  private formatEntry(
    entry: LogEntry,
    dated = false,
  ): {
    prefix: string;
    value: string;
    line: string;
    channelKey: string;
  } {
    const { channelNamesHidden, timestampsHidden, channelDataTypes, timeZone } =
      this.state;
    const { tsLen, configs } = this.internal;
    const chKeyStr = String(entry.channelKey);
    const cfg = configs[chKeyStr];
    const ts = !timestampsHidden
      ? new TimeStamp(entry.timestamp).toString("preciseTime", timeZone).slice(0, tsLen)
      : "";
    let value = entry.value;
    const isTimestampChannel = channelDataTypes[chKeyStr] === "timestamp";
    if (isTimestampChannel) {
      const { format, timeZone: valueTimeZone } = cfg?.timestampConfig ?? {
        format: "preciseDate",
        timeZone: "local",
      };
      try {
        value = new TimeStamp(BigInt(value)).toString(format, valueTimeZone);
      } catch {
        // Leave value as-is if it can't be parsed as a bigint.
      }
    } else if (cfg != null && (cfg.precision != null || cfg.notation !== "standard"))
      value = formatNumber(value, channelDataTypes[chKeyStr], cfg);
    const { displayNames, namePadding } = this.internal;
    const name = displayNames[chKeyStr] ?? chKeyStr;
    const pad = namePadding[chKeyStr] ?? "";
    let prefix: string;
    if (!timestampsHidden && !channelNamesHidden) prefix = `${ts}  ${name}${pad}   `;
    else if (!timestampsHidden) prefix = `${ts}   `;
    else if (!channelNamesHidden) prefix = `${name}${pad}   `;
    else prefix = "";
    if (dated && ts !== "") {
      const date = new TimeStamp(entry.timestamp).toString("ISODate", timeZone);
      prefix = `${date} ${prefix}`;
    }
    // Continuation lines (\n) keep the prefix's alignment width as whitespace.
    if (entry.continuation) prefix = " ".repeat(prefix.length);
    return { prefix, value, line: prefix + value, channelKey: chKeyStr };
  }

  selection(): Selection {
    const { selectionStart, selectionEnd } = this.state;
    if (selectionStart < 0 || selectionEnd < 0) return { lines: [], partial: false };
    const selMin = Math.min(selectionStart, selectionEnd);
    const selMax = Math.max(selectionStart, selectionEnd);
    const { valueColors, telem } = this.internal;
    const lines = this.entries.slice(selMin, selMax + 1).map((entry) => {
      const { line, channelKey } = this.formatEntry(entry, true);
      const lineColor = valueColors[channelKey];
      return { text: line, color: lineColor == null ? "" : color.hex(lineColor) };
    });
    const { exhausted, live } = telem.wrapped;
    const partial =
      selMin === 0 &&
      selMax === SELECT_ALL_END &&
      (exhausted === false || live === false);
    return { lines, partial };
  }

  private renderElements(draw2D: Draw2D, rows: Row[], lh: number): void {
    const reg = this.state.region;
    const font = this.state.font;
    const { prefixColors, defaultPrefixColor, textColor, valueColors, charWidth } =
      this.internal;
    const regTopX = box.left(reg);
    const regTopY = box.top(reg);
    for (let i = 0; i < rows.length; i++) {
      const { prefix, value, channelKey, first } = rows[i];
      const entryColor = valueColors[channelKey] ?? textColor;
      const prefixColor = prefixColors[channelKey] ?? defaultPrefixColor;
      const posX = regTopX + CONTENT_PADDING;
      const posY = regTopY + i * lh + CONTENT_PADDING;
      draw2D.text({
        text: prefix,
        level: font,
        color: prefixColor,
        position: { x: posX, y: posY },
        code: true,
      });
      const valueX = posX + prefix.length * charWidth;
      const isNegative = first && value[0] === "-";
      const displayValue = isNegative ? value.slice(1) : value;
      if (isNegative)
        draw2D.text({
          text: "-",
          level: font,
          color: entryColor,
          position: { x: valueX - charWidth, y: posY },
          code: true,
        });
      draw2D.text({
        text: displayValue,
        level: font,
        color: entryColor,
        position: { x: valueX, y: posY },
        code: true,
      });
    }
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Log.TYPE]: Log };
