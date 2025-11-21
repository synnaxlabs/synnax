// Copyright 2025 Synnax Labs, Inc.
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
  xy,
} from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { telem } from "@/telem/aether";
import { text } from "@/text/core";
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
});

const PADDING_Y = 6;
const PADDING_X = 6;
const SCROLLBAR_WIDTH = 6;
const VIEWPORT_WIDTH_TOLERANCE = 1;
const SCROLLBAR_RENDER_THRESHOLD = 0.98;
const CANVAS: render.Canvas2DVariant = "lower2d";

interface InternalState {
  theme: theming.Theme;
  render: render.Context;
  telem: telem.SeriesSource;
  textColor: color.Color;
  stopListeningTelem?: destructor.Destructor;
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

export class Log extends aether.Leaf<typeof logState, InternalState> {
  static readonly TYPE = "log";
  static readonly z = logState;
  schema = Log.z;
  values: MultiSeries = new MultiSeries([]);
  scrollState: ScrollbackState = ZERO_SCROLLBACK;
  private charWidth: number = 0;
  private wrappedCache: Map<number, string[]> = new Map();
  private totalVisualLines: number = 0;
  private cachedViewportWidth: number = 0;
  private cachedDataLength: number = 0;
  private cachedVisibleLogicalLineCount: number = 0;

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
      const off = this.values.alignmentBounds.upper - 1n;
      this.scrollState = {
        offset: off,
        offsetRef: off,
        scrollRef: this.state.wheelPos,
      };
      this.rebuildWrapCache();
      this.cachedDataLength = this.values.length;
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
        values.alignmentBounds.lower + BigInt(this.visibleLogicalLineCount)
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

    const previousLength = this.values.length;
    const [_, series] = this.internal.telem.value();
    this.values = series;

    if (this.values.length < previousLength || this.values.length === 0) {
      this.wrappedCache.clear();
      this.totalVisualLines = 0;
      this.cachedDataLength = 0;
      this.cachedVisibleLogicalLineCount = 0;
    }
    this.checkEmpty();
    i.stopListeningTelem?.();
    i.stopListeningTelem = i.telem.onChange(() => {
      const [_, series] = this.internal.telem.value();
      this.checkEmpty();
      this.values = series;
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
    return Math.ceil(this.totalVisualLines * this.lineHeight);
  }

  get visibleLineCount(): number {
    return Math.floor((box.height(this.state.region) - 12) / this.lineHeight);
  }

  get visibleLogicalLineCount(): number {
    return this.cachedVisibleLogicalLineCount;
  }

  private calculateAutoScrollStartIndex(): number {
    let visualLinesShown = 0;
    let logicalLinesNeeded = 0;
    const targetVisualLines = this.visibleLineCount - 1;
    for (let i = this.values.length - 1; i >= 0 && visualLinesShown < targetVisualLines; i--) {
      const wrappedLines = this.wrappedCache.get(i);
      if (wrappedLines) {
        visualLinesShown += wrappedLines.length;
        logicalLinesNeeded++;
      }
    }
    return Math.max(0, this.values.length - logicalLinesNeeded);
  }

  private shouldRebuildCache(currentWidth: number): boolean {
    if (this.charWidth === 0) return false;
    return (
      this.wrappedCache.size === 0 ||
      Math.abs(this.cachedViewportWidth - currentWidth) > VIEWPORT_WIDTH_TOLERANCE ||
      this.cachedDataLength !== this.values.length
    );
  }

  render(): render.Cleanup | undefined {
    const { render: renderCtx } = this.internal;
    const region = this.state.region;
    if (box.areaIsZero(region)) return undefined;
    if (!this.state.visible) return () => renderCtx.erase(region, xy.ZERO, CANVAS);
    const canvas = renderCtx[CANVAS];
    if (this.charWidth === 0) 
      this.charWidth = canvas.measureText('M').width;
    
    const lineMaxWidth = box.width(region) - (PADDING_X * 2) - SCROLLBAR_WIDTH;
    if (!this.state.scrolling && this.shouldRebuildCache(lineMaxWidth)) {
      this.rebuildWrapCache();
      this.cachedDataLength = this.values.length;
    }

    let range: Iterable<any>;
    let startLogicalIndex: number;

    if (!this.state.scrolling) {
      startLogicalIndex = this.calculateAutoScrollStartIndex();
      range = this.values.subIterator(startLogicalIndex, this.values.length);
    } else {
      // Calculate the start alignment for the visible window
      const start = this.values.traverseAlignment(
        this.scrollState.offset,
        -BigInt(this.visibleLogicalLineCount),
      );
      
      // Convert alignment to logical index by counting samples from lower bound
      startLogicalIndex = 0;
      for (const ser of this.values.series) 
        if (start < ser.alignment) break;
        else if (start >= ser.alignmentBounds.upper) startLogicalIndex += ser.length;
        else if (start >= ser.alignment && start < ser.alignmentBounds.upper) {
          startLogicalIndex += Number((start - ser.alignment) / ser.alignmentMultiple);
          break;
        }
      
      
      // Use regular iterator with the calculated indices
      const endLogicalIndex = Math.min(
        this.values.length, 
        startLogicalIndex + this.visibleLogicalLineCount
      );
      range = this.values.subIterator(startLogicalIndex, endLogicalIndex);
    }

    const reg = this.state.region;
    const draw2d = new Draw2D(canvas, this.internal.theme);
    const clearScissor = renderCtx.scissor(reg, xy.ZERO, [CANVAS]);
    this.renderElements(draw2d, range, startLogicalIndex);
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
    if (this.state.scrolling)
      scrollbarYPos -=
        (Number(
          this.values.distance(
            this.values.alignmentBounds.upper,
            this.scrollState.offset,
          ),
        ) /
          this.values.length) *
        box.height(reg);

    if (scrollbarYPos < 0) scrollbarYPos = box.top(reg);

    draw2d.container({
      region: box.construct(
        { x: box.right(reg) - SCROLLBAR_WIDTH, y: scrollbarYPos },
        { width: SCROLLBAR_WIDTH, height: scrollbarHeight },
      ),
      bordered: false,
      backgroundColor: (t: theming.Theme) => t.colors.gray.l6,
    });
  }

  private renderElements(
    draw2D: Draw2D, 
    iter: Iterable<TelemValue>,
    startLogicalIndex: number
  ): void {
    const reg = this.state.region;
    let visualLineIndex = 0;
    let logicalIndex = startLogicalIndex;

    for (const _value of iter) {
      const wrappedLines = this.wrappedCache.get(logicalIndex);
      if (!wrappedLines) {
        logicalIndex++;
        continue;
      }
      for (const line of wrappedLines) {
        const position = xy.translate(box.topLeft(reg), {
          x: PADDING_X,
          y: visualLineIndex * this.lineHeight + PADDING_Y
        });
        draw2D.text({
          text: line,
          level: this.state.font,
          shade: 11,
          position,
          code: true,
        });
        visualLineIndex++;
      }
      logicalIndex++;
    }
  }

  private softWrapLog(text: string, maxWidth: number): string[] {
    const lines: string[] = [];
    // Collapse exit early if/else
    if (text.length === 0) return [""];
    if (this.charWidth === 0) return [text];
    if (text.length * this.charWidth <= maxWidth) return [text];
    let currentLine: string = "";
    let currentWidth: number = 0;
    const charsPerLine = Math.floor(maxWidth / this.charWidth);
    // Hot path: no words, wrap by character
    if (!text.includes(" "))
      return Array.from(
        { length: Math.ceil(text.length / charsPerLine) },
        (_, i) => text.slice(i * charsPerLine, (i + 1) * charsPerLine)
      );

    const words = text.split(" ");
    const spaceWidth = this.charWidth;
    for (const word of words) {
      const wordWidth = word.length * this.charWidth;
      if (wordWidth > maxWidth) {
        if (currentLine !== "") {
          lines.push(currentLine);
          currentLine = "";
          currentWidth = 0;
        }
        for (let i = 0; i < word.length; i += charsPerLine) {
          const chunk = word.slice(i, i + charsPerLine);
          if (i + charsPerLine < word.length) 
            lines.push(chunk);
           else {
            currentLine = chunk;
            currentWidth = chunk.length * this.charWidth;
          }
        }
        continue;
      }
      if (currentLine === "") {
        currentLine = word;
        currentWidth = wordWidth;
      } else {
        const widthWithWord = currentWidth + spaceWidth + wordWidth;
        if (widthWithWord <= maxWidth) {
          currentLine += ` ${  word}`;
          currentWidth = widthWithWord;
        } else {
          lines.push(currentLine);
          currentLine = word;
          currentWidth = wordWidth;
        }
      }
    }
    if (currentLine !== "") 
      lines.push(currentLine);
    
    return lines;
  }

  private rebuildWrapCache(): void {
    const lineMaxWidth = box.width(this.state.region) - (PADDING_X * 2) - SCROLLBAR_WIDTH;
    this.wrappedCache.clear();
    this.totalVisualLines = 0;
    if (this.charWidth === 0 || lineMaxWidth <= 0) {
      this.cachedViewportWidth = lineMaxWidth;
      this.cachedVisibleLogicalLineCount = 0;
      return;
    }
    const isJson = this.values.dataType.equals(DataType.JSON);
    let index = 0;
    for (const value of this.values) {
      if (value == null) {
        index++;
        continue;
      }
      const text = isJson ? JSON.stringify(value) : value.toString();
      const wrappedLines = this.softWrapLog(text, lineMaxWidth);
      this.wrappedCache.set(index, wrappedLines);
      this.totalVisualLines += wrappedLines.length;
      index++;
    }
    this.cachedViewportWidth = lineMaxWidth;
    let visualLinesShown = 0;
    let logicalLinesNeeded = 0;
    const targetVisualLines = this.visibleLineCount;
    for (let i = this.values.length - 1; i >= 0 && visualLinesShown < targetVisualLines; i--) {
      const wrappedLines = this.wrappedCache.get(i);
      if (wrappedLines) {
        visualLinesShown += wrappedLines.length;
        logicalLinesNeeded++;
      }
    }
    this.cachedVisibleLogicalLineCount = logicalLinesNeeded;
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Log.TYPE]: Log };
