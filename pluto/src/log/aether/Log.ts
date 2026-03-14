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
  telem: telem.logSourceSpecZ.default(telem.noopLogSourceSpec),
  font: text.levelZ.default("p"),
  color: color.colorZ.default(color.ZERO),
  overshoot: xy.xy.default({ x: 0, y: 0 }),
  selectionStart: z.number().default(-1),
  selectionEnd: z.number().default(-1),
  visibleStart: z.number().default(0),
  selectedText: z.string().default(""),
  selectedLines: z
    .array(z.object({ text: z.string(), color: z.string() }))
    .default([]),
  computedLineHeight: z.number().default(0),
  copyFlash: z.boolean().default(false),
});

const SCROLLBAR_RENDER_THRESHOLD = 0.98;
const CANVAS: render.Canvas2DVariant = "lower2d";
const CONTENT_PADDING = 6;

interface InternalState {
  theme: theming.Theme;
  render: render.Context;
  telem: telem.LogSource;
  textColor: color.Color;
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

export class Log extends aether.Leaf<typeof logState, InternalState> {
  static readonly TYPE = "log";
  static readonly z = logState;
  schema = Log.z;
  entries: LogEntry[] = [];
  scrollState: ScrollbackState = ZERO_SCROLLBACK;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.render = render.Context.use(ctx);
    i.theme = theming.use(ctx);
    if (color.isZero(this.state.color))
      this.internal.textColor = i.theme.colors.gray.l11;
    else i.textColor = this.state.color;
    i.telem = telem.useSource(ctx, this.state.telem, i.telem);

    const { scrolling, wheelPos } = this.state;

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
      const dist = Math.ceil((wheelPos - scrollState.scrollRef) / this.lineHeight);
      scrollState.offset = Math.max(
        this.visibleLineCount,
        Math.min(scrollState.offsetRef - dist, this.entries.length),
      );
      // Scrolled to the very top
      if (scrollState.offset <= this.visibleLineCount) {
        scrollState.offset = this.visibleLineCount;
        this.setState((s) => ({ ...s, wheelPos: this.prevState.wheelPos }));
      }
      // Scrolled back to live
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
            this.visibleLineCount,
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
    if (actuallyEmpty === this.state.empty) return;
    this.setState((s) => ({ ...s, empty: actuallyEmpty }));
  }

  afterDelete(): void {
    const { telem, render: renderCtx } = this.internal;
    telem.cleanup?.();
    renderCtx.erase(box.construct(this.state.region), xy.ZERO, CANVAS);
  }

  private requestRender(): void {
    const { render } = this.internal;
    render.loop.set({
      key: `${this.type}-${this.key}`,
      render: () => this.render(),
      priority: "high",
      canvases: [CANVAS],
    });
  }

  get lineHeight(): number {
    return (
      this.internal.theme.typography[this.state.font].size *
      this.internal.theme.sizes.base
    );
  }

  get totalHeight(): number {
    return Math.ceil(this.entries.length * this.lineHeight);
  }

  get visibleLineCount(): number {
    return Math.min(
      Math.floor(
        (box.height(this.state.region) - CONTENT_PADDING * 2) / this.lineHeight,
      ),
      this.entries.length,
    );
  }

  render(): render.Cleanup | undefined {
    const { render: renderCtx } = this.internal;
    const region = this.state.region;
    if (box.areaIsZero(region)) return undefined;
    if (!this.state.visible) return () => renderCtx.erase(region, xy.ZERO, CANVAS);

    const visible = this.visibleLineCount;
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

    const lh = this.lineHeight;
    if (this.state.visibleStart !== sliceStart || this.state.computedLineHeight !== lh)
      this.setState((s) => ({ ...s, visibleStart: sliceStart, computedLineHeight: lh }));

    this.updateSelectedText();

    const reg = this.state.region;
    const canvas = renderCtx[CANVAS];
    const draw2d = new Draw2D(canvas, this.internal.theme);
    const clearScissor = renderCtx.scissor(reg, xy.ZERO, [CANVAS]);
    this.renderSelection(draw2d, sliceStart, slice.length);
    this.renderElements(draw2d, slice);
    this.renderScrollbar(draw2d);
    clearScissor();
    const eraseRegion = box.copy(this.state.region);
    return ({ canvases }) =>
      renderCtx.erase(eraseRegion, this.state.overshoot, ...canvases);
  }

  private renderScrollbar(draw2d: Draw2D): void {
    const reg = this.state.region;
    const scrollbarHeight = (box.height(reg) / this.totalHeight) * box.height(reg);
    if (scrollbarHeight >= box.height(reg) * SCROLLBAR_RENDER_THRESHOLD) return;
    let scrollbarYPos = box.bottom(reg) - scrollbarHeight;
    if (this.state.scrolling) {
      const distFromEnd = this.entries.length - this.scrollState.offset;
      scrollbarYPos -= (distFromEnd / this.entries.length) * box.height(reg);
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

  private renderSelection(draw2d: Draw2D, sliceStart: number, visibleCount: number): void {
    const { selectionStart, selectionEnd } = this.state;
    if (selectionStart < 0 || selectionEnd < 0) return;
    const selMin = Math.min(selectionStart, selectionEnd);
    const selMax = Math.max(selectionStart, selectionEnd);
    const sliceEnd = sliceStart + visibleCount;
    if (selMax < sliceStart || selMin >= sliceEnd) return;
    const reg = this.state.region;
    const lh = this.lineHeight;
    const highlightStart = Math.max(selMin, sliceStart) - sliceStart;
    const highlightEnd = Math.min(selMax, sliceEnd - 1) - sliceStart;
    const selColor = color.setAlpha(this.internal.theme.colors.primary.z, 0.25);
    const flashColor = color.setAlpha(this.internal.theme.colors.primary.z, 0.15);
    const bgColor = this.state.copyFlash ? flashColor : selColor;
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
  private formatEntry(
    entry: LogEntry,
  ): { line: string; cfg: z.infer<typeof channelConfigZ> | undefined } {
    const { showChannelNames, timestampPrecision, channelConfigs } = this.state;
    const tsLen = timestampPrecision === 0 ? 8 : 9 + timestampPrecision;
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
    const line = showChannelNames
      ? `${ts}  [${entry.channelName}]${entry.channelPadding}  ${value}`
      : `${ts}  ${value}`;
    return { line, cfg };
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
      const selectedLines = formatted.map((f) => ({
        text: f.line,
        color: f.cfg?.color ?? "",
      }));
      this.setState((s) => ({ ...s, selectedText: text, selectedLines }));
    }
  }

  private renderElements(draw2D: Draw2D, entries: LogEntry[]): void {
    const reg = this.state.region;
    for (let i = 0; i < entries.length; i++) {
      const { line, cfg } = this.formatEntry(entries[i]);
      draw2D.text({
        text: line,
        level: this.state.font,
        shade: cfg?.color ? undefined : 11,
        color: cfg?.color ? cfg.color : undefined,
        position: xy.translate(box.topLeft(reg), {
          x: CONTENT_PADDING,
          y: i * this.lineHeight + CONTENT_PADDING,
        }),
        code: true,
      });
    }
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Log.TYPE]: Log };
