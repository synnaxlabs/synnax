// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, color, type destructor, notation, TimeStamp, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { telem } from "@/telem/aether";
import { type LogEntry } from "@/telem/aether/telem";
import { text } from "@/text/base";
import { theming } from "@/theming/aether";
import { Draw2D } from "@/vis/draw2d";
import { render } from "@/vis/render";

export const channelConfigZ = z.object({
  color: z.string().default(""),
  notation: notation.notationZ.default("standard"),
  precision: z.number().min(-1).max(17).default(-1),
  alias: z.string().default(""),
});

export const logState = z.object({
  region: box.box,
  wheelPos: z.number(),
  scrolling: z.boolean(),
  empty: z.boolean(),
  visible: z.boolean(),
  showChannelNames: z.boolean().default(true),
  timestampPrecision: z.number().min(0).max(3).default(0),
  channelConfigs: z.record(z.string(), channelConfigZ).default({}),
  channels: z.array(z.number().or(z.string())).default([]),
  telem: telem.logSourceSpecZ.default(telem.noopLogSourceSpec),
  font: text.levelZ.default("p"),
  color: color.colorZ.default(color.ZERO),
  overshoot: xy.xy.default({ x: 0, y: 0 }),
  selectionStart: z.number().default(-1),
  selectionEnd: z.number().default(-1),
  visibleStart: z.number().default(0),
  selectedText: z.string().default(""),
  selectedLines: z.array(z.object({ text: z.string(), color: z.string() })).default([]),
  computedLineHeight: z.number().default(0),
  entryCount: z.number().default(0),
  copyFlash: z.boolean().default(false),
});

const SCROLLBAR_RENDER_THRESHOLD = 0.98;
const CANVAS: render.Canvas2DVariant = "lower2d";
const CONTENT_PADDING = 6;

// Per-theme prefix color muting — multiplied with the base color's HSLA.
const DARK_PREFIX = { hue: 1, saturation: 0.8, lightness: 0.85, alpha: 0.95 };
const LIGHT_PREFIX = { hue: 1, saturation: 0.8, lightness: 0.8, alpha: 0.8 };

interface InternalState {
  theme: theming.Theme;
  render: render.Context;
  draw2d: Draw2D;
  telem: telem.LogSource;
  textColor: color.Color;
  prefixColors: Record<string, color.Color>;
  defaultPrefixColor: color.Color;
  // Cached per-channel value colors to avoid repeated lookups in render loop.
  valueColors: Record<string, color.Color>;
  charWidth: number;
  lineHeight: number;
  tsLen: number;
  selectionColor: color.Color;
  selectionFlashColor: color.Color;
  stopListeningTelem?: destructor.Destructor;
}

interface ScrollbackState {
  offset: number;
  offsetRef: number;
  scrollRef: number;
}

const ZERO_SCROLLBACK: ScrollbackState = {
  offset: 0,
  offsetRef: 0,
  scrollRef: 0,
};

const muteColor = (c: color.Crude, theme: theming.Theme): color.Color => {
  const p = theme.key === "synnaxDark" ? DARK_PREFIX : LIGHT_PREFIX;
  const [h, s, l, a] = color.hsla(c);
  return color.fromHSLA([
    Math.min(360, Math.round(h * p.hue)),
    Math.min(100, Math.round(s * p.saturation)),
    Math.min(100, Math.round(l * p.lightness)),
    Math.min(1, a * p.alpha),
  ]);
};

export class Log extends aether.Leaf<typeof logState, InternalState> {
  static readonly TYPE = "log";
  static readonly z = logState;
  schema = Log.z;
  entries: LogEntry[] = [];
  scrollState: ScrollbackState = ZERO_SCROLLBACK;
  // Render key for requestRender — allocated once, not per call.
  private renderKey: string = "";

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
    i.render = render.Context.use(ctx);
    i.theme = theming.use(ctx);
    i.draw2d = new Draw2D(i.render[CANVAS], i.theme);
    if (this.renderKey === "") this.renderKey = `${this.type}-${this.key}`;

    if (color.isZero(this.state.color)) i.textColor = i.theme.colors.gray.l11;
    else i.textColor = this.state.color;

    i.lineHeight = i.theme.typography[this.state.font].size * i.theme.sizes.base;
    i.charWidth = i.draw2d.measureCharWidth(this.state.font);
    i.tsLen =
      this.state.timestampPrecision === 0 ? 8 : 9 + this.state.timestampPrecision;

    // Rebuild color caches only when configs or base color changed.
    const configs = this.state.channelConfigs;
    const prevConfigs = this.prevState.channelConfigs;
    const colorChanged = !color.equals(this.state.color, this.prevState.color);
    if (configs !== prevConfigs || colorChanged || i.prefixColors == null) {
      i.defaultPrefixColor = muteColor(i.textColor, i.theme);
      i.prefixColors = {};
      i.valueColors = {};
      for (const [key, cfg] of Object.entries(configs))
        if (cfg.color) {
          i.prefixColors[key] = muteColor(cfg.color, i.theme);
          i.valueColors[key] = color.construct(cfg.color);
        }
    }

    // Cache selection highlight colors (theme-dependent).
    i.selectionColor = color.setAlpha(i.theme.colors.primary.z, 0.25);
    i.selectionFlashColor = color.setAlpha(i.theme.colors.primary.z, 0.15);

    i.telem = telem.useSource(ctx, this.state.telem, i.telem);

    // Always call setChannels — the source short-circuits if unchanged.
    // This handles both initial setup (where prevState === state) and subsequent changes.
    i.telem.setChannels?.(this.state.channels);

    if (configs !== prevConfigs) {
      const aliases: Record<string, string> = {};
      for (const [key, cfg] of Object.entries(configs))
        if (cfg.alias) aliases[key] = cfg.alias;
      i.telem.setAliases?.(aliases);
    }

    const { scrolling, wheelPos } = this.state;
    const lh = i.lineHeight;

    const justEnteredScrollback = this.state.scrolling && !this.prevState.scrolling;
    if (justEnteredScrollback) {
      const off = this.entries.length;
      this.scrollState = {
        offset: off,
        offsetRef: off,
        scrollRef: this.state.wheelPos,
      };
    } else if (scrolling) {
      const { scrollState } = this;
      const dist = Math.ceil((wheelPos - scrollState.scrollRef) / lh);
      const visCount = this.calcVisibleLineCount(lh);
      scrollState.offset = Math.max(
        visCount,
        Math.min(scrollState.offsetRef - dist, this.entries.length),
      );
      if (scrollState.offset <= visCount) {
        scrollState.offset = visCount;
        this.setState((s) => ({ ...s, wheelPos: this.prevState.wheelPos }));
      }
      if (scrollState.offset >= this.entries.length)
        this.setState((s) => ({ ...s, scrolling: false }));
    }

    this.entries = this.internal.telem.value();
    this.checkEmpty();
    i.stopListeningTelem?.();
    i.stopListeningTelem = i.telem.onChange(() => {
      const { evictedCount } = this.internal.telem;
      this.entries = this.internal.telem.value();
      if (evictedCount > 0) {
        if (this.state.scrolling)
          this.scrollState.offset = Math.max(
            this.calcVisibleLineCount(this.internal.lineHeight),
            this.scrollState.offset - evictedCount,
          );
        this.clampSelection(evictedCount);
      }
      this.checkEmpty();
      this.requestRender();
    });
    if (!this.state.visible && !this.prevState.visible) return;
    this.requestRender();
  }

  private checkEmpty(): void {
    const actuallyEmpty = this.entries.length === 0;
    const countChanged = this.entries.length !== this.state.entryCount;
    if (actuallyEmpty === this.state.empty && !countChanged) return;
    this.setState((s) => ({
      ...s,
      empty: actuallyEmpty,
      entryCount: this.entries.length,
    }));
  }

  afterDelete(): void {
    const { telem, render: renderCtx } = this.internal;
    telem.cleanup?.();
    renderCtx.erase(box.construct(this.state.region), xy.ZERO, CANVAS);
  }

  private requestRender(): void {
    this.internal.render.loop.set({
      key: this.renderKey,
      render: () => this.render(),
      priority: "high",
      canvases: [CANVAS],
    });
  }

  private calcVisibleLineCount(lh: number): number {
    return Math.min(
      Math.floor((box.height(this.state.region) - CONTENT_PADDING * 2) / lh),
      this.entries.length,
    );
  }

  render(): render.Cleanup | undefined {
    const { render: renderCtx, lineHeight: lh, draw2d } = this.internal;
    const region = this.state.region;
    if (box.areaIsZero(region)) return undefined;
    if (!this.state.visible) return () => renderCtx.erase(region, xy.ZERO, CANVAS);

    const visible = this.calcVisibleLineCount(lh);
    let sliceStart: number;
    let slice: LogEntry[];
    if (!this.state.scrolling) {
      sliceStart = Math.max(0, this.entries.length - visible);
      slice = this.entries.slice(sliceStart);
    } else {
      const end = this.scrollState.offset;
      sliceStart = Math.max(0, end - visible);
      slice = this.entries.slice(sliceStart, end);
    }

    if (this.state.visibleStart !== sliceStart || this.state.computedLineHeight !== lh)
      this.setState((s) => ({
        ...s,
        visibleStart: sliceStart,
        computedLineHeight: lh,
      }));

    this.updateSelectedText();

    const reg = this.state.region;
    const clearScissor = renderCtx.scissor(reg, xy.ZERO, [CANVAS]);
    this.renderSelection(draw2d, sliceStart, slice.length, lh);
    this.renderElements(draw2d, slice, lh);
    this.renderScrollbar(draw2d, lh);
    clearScissor();
    const eraseRegion = box.copy(this.state.region);
    return ({ canvases }) =>
      renderCtx.erase(eraseRegion, this.state.overshoot, ...canvases);
  }

  private renderScrollbar(draw2d: Draw2D, lh: number): void {
    if (!this.state.scrolling) return;
    const reg = this.state.region;
    const totalHeight = Math.ceil(this.entries.length * lh);
    const regHeight = box.height(reg);
    const scrollbarHeight = (regHeight / totalHeight) * regHeight;
    if (scrollbarHeight >= regHeight * SCROLLBAR_RENDER_THRESHOLD) return;
    let scrollbarYPos = box.bottom(reg) - scrollbarHeight;
    if (this.state.scrolling) {
      const distFromEnd = this.entries.length - this.scrollState.offset;
      scrollbarYPos -= (distFromEnd / this.entries.length) * regHeight;
    }
    if (scrollbarYPos < 0) scrollbarYPos = box.top(reg);
    draw2d.container({
      region: box.construct(
        { x: box.right(reg) - 6, y: scrollbarYPos },
        { width: 6, height: scrollbarHeight },
      ),
      bordered: false,
      backgroundColor: (t: theming.Theme) => t.colors.gray.l6,
    });
  }

  private clampSelection(evictedCount: number): void {
    const { selectionStart, selectionEnd } = this.state;
    if (selectionStart < 0) return;
    const newStart = selectionStart - evictedCount;
    const newEnd = selectionEnd - evictedCount;
    if (newEnd < 0)
      this.setState((s) => ({
        ...s,
        selectionStart: -1,
        selectionEnd: -1,
        selectedText: "",
      }));
    else
      this.setState((s) => ({
        ...s,
        selectionStart: Math.max(0, newStart),
        selectionEnd: newEnd,
        selectedText: "",
      }));
  }

  private renderSelection(
    draw2d: Draw2D,
    sliceStart: number,
    visibleCount: number,
    lh: number,
  ): void {
    const { selectionStart, selectionEnd } = this.state;
    if (selectionStart < 0 || selectionEnd < 0) return;
    const selMin = Math.min(selectionStart, selectionEnd);
    const selMax = Math.max(selectionStart, selectionEnd);
    const sliceEnd = sliceStart + visibleCount;
    if (selMax < sliceStart || selMin >= sliceEnd) return;
    const reg = this.state.region;
    const highlightStart = Math.max(selMin, sliceStart) - sliceStart;
    const highlightEnd = Math.min(selMax, sliceEnd - 1) - sliceStart;
    const bgColor = this.state.copyFlash
      ? this.internal.selectionFlashColor
      : this.internal.selectionColor;
    const rowCount = highlightEnd - highlightStart + 1;
    draw2d.container({
      region: box.construct(
        xy.translate(box.topLeft(reg), {
          x: 0,
          y: highlightStart * lh + CONTENT_PADDING,
        }),
        { width: box.width(reg), height: rowCount * lh },
      ),
      bordered: false,
      rounded: false,
      backgroundColor: bgColor,
    });
  }

  // showChannelNames is read from state (O(1)) rather than derived by scanning all
  // entries (O(n)). The render loop below is already O(n) over visible entries —
  // adding a second O(n) scan here just to answer a yes/no question would double the
  // per-frame work at up to 60fps.
  private formatEntry(entry: LogEntry): {
    prefix: string;
    value: string;
    line: string;
    channelKey: string;
  } {
    const { showChannelNames, channelConfigs } = this.state;
    const { tsLen } = this.internal;
    const cfg = channelConfigs[String(entry.channelKey)];
    const ts = new TimeStamp(entry.timestamp)
      .toString("preciseTime", "local")
      .slice(0, tsLen);
    let value = entry.value;
    if (cfg != null && (cfg.precision >= 0 || cfg.notation !== "standard")) {
      const num = parseFloat(value);
      if (!isNaN(num)) {
        const precision = cfg.precision >= 0 ? cfg.precision : 0;
        value = notation.stringifyNumber(num, precision, cfg.notation);
      }
    }
    const prefix = showChannelNames
      ? `${ts}  [${entry.channelName}]${entry.channelPadding}  `
      : `${ts}  `;
    return {
      prefix,
      value,
      line: prefix + value,
      channelKey: String(entry.channelKey),
    };
  }

  private updateSelectedText(): void {
    const { selectionStart, selectionEnd } = this.state;
    if (selectionStart < 0 || selectionEnd < 0) {
      if (this.state.selectedText !== "")
        this.setState((s) => ({ ...s, selectedText: "", selectedLines: [] }));
      return;
    }
    const selMin = Math.min(selectionStart, selectionEnd);
    const selMax = Math.max(selectionStart, selectionEnd);
    const selected = this.entries.slice(selMin, selMax + 1);
    const formatted = selected.map((e) => this.formatEntry(e));
    const text = formatted.map((f) => f.line).join("\n");
    if (text !== this.state.selectedText) {
      const { valueColors } = this.internal;
      const selectedLines = formatted.map((f) => ({
        text: f.line,
        color: f.channelKey in valueColors ? color.hex(valueColors[f.channelKey]) : "",
      }));
      this.setState((s) => ({ ...s, selectedText: text, selectedLines }));
    }
  }

  private renderElements(draw2D: Draw2D, entries: LogEntry[], lh: number): void {
    const reg = this.state.region;
    const font = this.state.font;
    const { prefixColors, defaultPrefixColor, textColor, valueColors, charWidth } =
      this.internal;
    const regTopX = box.left(reg);
    const regTopY = box.top(reg);
    for (let i = 0; i < entries.length; i++) {
      const { prefix, value, channelKey } = this.formatEntry(entries[i]);
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
      draw2D.text({
        text: value,
        level: font,
        color: entryColor,
        position: { x: posX + prefix.length * charWidth, y: posY },
        code: true,
      });
    }
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Log.TYPE]: Log };
