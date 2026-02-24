// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  box,
  color,
  DataType,
  type destructor,
  MultiSeries,
  type TelemValue,
  TimeStamp,
  xy,
} from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { telem } from "@/telem/aether";
import { text } from "@/text/base";
import { theming } from "@/theming/aether";
import { Draw2D } from "@/vis/draw2d";
import { render } from "@/vis/render";

export const logState = z.object({
  region: box.box,
  wheelPos: z.number(),
  scrolling: z.boolean(),
  empty: z.boolean(),
  visible: z.boolean(),
  telem: telem.seriesSourceSpecZ.default(telem.noopSeriesSourceSpec),
  font: text.levelZ.default("p"),
  color: color.colorZ.default(color.ZERO),
  overshoot: xy.xy.default({ x: 0, y: 0 }),
  lineCount: z.number().default(0),
  indexTelem: telem.seriesSourceSpecZ.default(telem.noopSeriesSourceSpec),
  showIndex: z.boolean().default(false),
});

export const logMethodsZ = {
  copyText: z.function({
    input: z.tuple([z.number(), z.number()]),
    output: z.string(),
  }),
  copyAllText: z.function({ input: z.tuple([]), output: z.string() }),
};

const SCROLLBAR_RENDER_THRESHOLD = 0.98;
const MIN_SCROLLBAR_HEIGHT = 20;
const MAX_COPY_LINES = 100_000;
const CANVAS: render.Canvas2DVariant = "lower2d";

interface InternalState {
  theme: theming.Theme;
  render: render.Context;
  telem: telem.SeriesSource;
  textColor: color.Color;
  stopListeningTelem?: destructor.Destructor;
  indexTelem: telem.SeriesSource;
  stopListeningIndexTelem?: destructor.Destructor;
}

interface ScrollbackState {
  offset: bigint;
  offsetRef: bigint;
  scrollRef: number;
}

const ZERO_SCROLLBACK: ScrollbackState = {
  offset: 0n,
  offsetRef: 0n,
  scrollRef: 0,
};

interface RenderCacheKey {
  startIdx: number;
  endIdx: number;
  valuesLen: number;
  showIndex: boolean;
  indexLen: number;
}

export class Log
  extends aether.Leaf<typeof logState, InternalState, typeof logMethodsZ>
  implements aether.HandlersFromSchema<typeof logMethodsZ>
{
  static readonly TYPE = "log";
  static readonly z = logState;
  static readonly METHODS = logMethodsZ;
  schema = Log.z;
  methods = logMethodsZ;
  values: MultiSeries = new MultiSeries([]);
  indexValues: MultiSeries = new MultiSeries([]);
  scrollState: ScrollbackState = ZERO_SCROLLBACK;
  private renderCache: string[] = [];
  private renderCacheKey: RenderCacheKey | null = null;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.render = render.Context.use(ctx);
    i.theme = theming.use(ctx);
    if (color.isZero(this.state.color))
      this.internal.textColor = i.theme.colors.gray.l11;
    else i.textColor = this.state.color;
    i.telem = telem.useSource(ctx, this.state.telem, i.telem);
    i.indexTelem = telem.useSource(ctx, this.state.indexTelem, i.indexTelem);

    const { scrolling, wheelPos } = this.state;

    const justEnteredScrollback = this.state.scrolling && !this.prevState.scrolling;
    if (justEnteredScrollback) {
      const off = this.values.alignmentBounds.upper - 1n;
      this.scrollState = {
        offset: off,
        offsetRef: off,
        scrollRef: this.state.wheelPos,
      };
    } else if (scrolling) {
      const { scrollState, values } = this;
      const dist = Math.ceil((wheelPos - this.scrollState.scrollRef) / this.lineHeight);
      scrollState.offset = this.values.traverseAlignment(
        scrollState.offsetRef,
        -BigInt(dist),
      );
      // This means that the last element is visible at the top of the viewport, so we
      // should stop scrolling.
      if (
        scrollState.offset <
        values.alignmentBounds.lower + BigInt(this.visibleLineCount)
      ) {
        scrollState.offset = values.alignmentBounds.lower;
        // Set the wheel position back to it's previous location so we can scroll back
        // down without jumping.
        this.setState((s) => ({ ...s, wheelPos: this.prevState.wheelPos }));
      }
      // If we've scrolled back to the bottom fo the log, stop scrolling and go back
      // to live mode.
      if (scrollState.offset >= values.alignmentBounds.upper)
        this.setState((s) => ({ ...s, scrolling: false }));
    }

    const [_, series] = this.internal.telem.value();
    this.values = series;
    const [__, indexSeries] = this.internal.indexTelem.value();
    this.indexValues = indexSeries;
    this.checkEmpty();
    this.updateLineCount();

    i.stopListeningTelem?.();
    i.stopListeningTelem = i.telem.onChange(() => {
      const [_, series] = this.internal.telem.value();
      this.checkEmpty();
      this.values = series;
      this.updateLineCount();
      this.requestRender();
    });
    i.stopListeningIndexTelem?.();
    i.stopListeningIndexTelem = i.indexTelem.onChange(() => {
      const [_, series] = this.internal.indexTelem.value();
      this.indexValues = series;
      this.requestRender();
    });
    if (!this.state.visible && !this.prevState.visible) return;
    this.requestRender();
  }

  private checkEmpty(): void {
    const actuallyEmpty = this.values.length === 0;
    if (actuallyEmpty === this.state.empty) return;
    this.setState((s) => ({ ...s, empty: actuallyEmpty }));
  }

  private updateLineCount(): void {
    const lineCount = Math.min(this.visibleLineCount, this.values.length);
    if (lineCount !== this.state.lineCount)
      this.setState((s) => ({ ...s, lineCount }));
  }

  afterDelete(): void {
    const { telem, indexTelem, render: renderCtx } = this.internal;
    telem.cleanup?.();
    indexTelem.cleanup?.();
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
    return Math.ceil(this.values.length * this.lineHeight);
  }

  get visibleLineCount(): number {
    return Math.min(
      Math.floor((box.height(this.state.region) - 12) / this.lineHeight),
      this.values.length,
    );
  }

  private valueToText(value: TelemValue): string {
    return this.values.dataType.equals(DataType.JSON)
      ? JSON.stringify(value)
      : value.toString();
  }

  private formatLine(value: TelemValue, indexValue?: TelemValue): string {
    const text = this.valueToText(value);
    if (!this.state.showIndex || indexValue == null) return text;
    const ts = new TimeStamp(BigInt(indexValue as number));
    return `[${ts.toString("preciseTime")}] ${text}`;
  }

  private getVisibleRange(): { startIdx: number; endIdx: number } {
    if (!this.state.scrolling) {
      const startIdx = this.values.length - this.visibleLineCount;
      return { startIdx, endIdx: this.values.length };
    }
    const start = this.values.traverseAlignment(
      this.scrollState.offset,
      -BigInt(this.visibleLineCount),
    );
    const startIdx = Number(start - this.values.alignmentBounds.lower);
    return { startIdx, endIdx: startIdx + this.visibleLineCount };
  }

  private resolveLines(startIdx: number, endIdx: number): string[] {
    const lines: string[] = [];
    const iter = this.values.subIterator(startIdx, endIdx);
    let indexIter: Iterator<TelemValue> | undefined;
    if (this.state.showIndex && this.indexValues.length > 0) {
      const iterable = this.indexValues.subIterator(startIdx, endIdx);
      indexIter = iterable[Symbol.iterator]();
    }
    for (const value of iter) {
      const indexValue = indexIter?.next().value;
      lines.push(this.formatLine(value, indexValue));
    }
    return lines;
  }

  private getVisibleLines(): string[] {
    const { startIdx, endIdx } = this.getVisibleRange();
    const key: RenderCacheKey = {
      startIdx,
      endIdx,
      valuesLen: this.values.length,
      showIndex: this.state.showIndex,
      indexLen: this.indexValues.length,
    };
    const prev = this.renderCacheKey;
    if (
      prev != null &&
      prev.startIdx === key.startIdx &&
      prev.endIdx === key.endIdx &&
      prev.valuesLen === key.valuesLen &&
      prev.showIndex === key.showIndex &&
      prev.indexLen === key.indexLen
    )
      return this.renderCache;
    this.renderCache = this.resolveLines(startIdx, endIdx);
    this.renderCacheKey = key;
    return this.renderCache;
  }

  copyText(startLine: number, endLine: number): string {
    const { startIdx } = this.getVisibleRange();
    return this.resolveLines(startIdx + startLine, startIdx + endLine).join("\n");
  }

  copyAllText(): string {
    const total = this.values.length;
    const count = Math.min(total, MAX_COPY_LINES);
    const startIdx = total - count;
    const lines = this.resolveLines(startIdx, total);
    if (total > MAX_COPY_LINES)
      lines.unshift(`... (${total - MAX_COPY_LINES} earlier lines omitted)`);
    return lines.join("\n");
  }

  render(): render.Cleanup | undefined {
    const { render: renderCtx } = this.internal;
    const region = this.state.region;
    if (box.areaIsZero(region)) return undefined;
    if (!this.state.visible) return () => renderCtx.erase(region, xy.ZERO, CANVAS);

    const lines = this.getVisibleLines();
    const reg = this.state.region;
    const canvas = renderCtx[CANVAS];
    const draw2d = new Draw2D(canvas, this.internal.theme);
    const clearScissor = renderCtx.scissor(reg, xy.ZERO, [CANVAS]);
    this.renderElements(draw2d, lines);
    this.renderScrollbar(draw2d);
    clearScissor();
    const eraseRegion = box.copy(this.state.region);
    return ({ canvases }) =>
      renderCtx.erase(eraseRegion, this.state.overshoot, ...canvases);
  }

  private renderScrollbar(draw2d: Draw2D): void {
    const reg = this.state.region;
    const viewportH = box.height(reg);
    const rawHeight = (viewportH / this.totalHeight) * viewportH;
    if (rawHeight >= viewportH * SCROLLBAR_RENDER_THRESHOLD) return;
    const scrollbarHeight = Math.max(rawHeight, MIN_SCROLLBAR_HEIGHT);
    const top = box.top(reg);
    const maxTravel = viewportH - scrollbarHeight;
    let scrollbarYPos: number;
    if (!this.state.scrolling) scrollbarYPos = top + maxTravel;
    else {
      const scrollFraction =
        Number(
          this.values.distance(
            this.values.alignmentBounds.upper,
            this.scrollState.offset,
          ),
        ) / this.values.length;
      scrollbarYPos = top + maxTravel * (1 - scrollFraction);
    }
    scrollbarYPos = Math.max(top, Math.min(scrollbarYPos, top + maxTravel));

    draw2d.container({
      region: box.construct(
        { x: box.right(reg) - 6, y: scrollbarYPos },
        { width: 6, height: scrollbarHeight },
      ),
      bordered: false,
      backgroundColor: (t: theming.Theme) => t.colors.gray.l6,
    });
  }

  private renderElements(draw2D: Draw2D, lines: string[]): void {
    const reg = this.state.region;
    for (let i = 0; i < lines.length; i++) 
      draw2D.text({
        text: lines[i],
        level: this.state.font,
        shade: 11,
        position: xy.translate(box.topLeft(reg), { x: 6, y: i * this.lineHeight + 6 }),
        code: true,
      });
    
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Log.TYPE]: Log };
