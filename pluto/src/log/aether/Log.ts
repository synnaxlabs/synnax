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
  overshoot: xy.xyZ.default({ x: 0, y: 0 }),
});

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
    return Math.ceil(this.values.length * this.lineHeight);
  }

  get visibleLineCount(): number {
    return Math.min(
      Math.floor((box.height(this.state.region) - 12) / this.lineHeight),
      this.values.length,
    );
  }

  render(): render.Cleanup | undefined {
    const { render: renderCtx } = this.internal;
    const region = this.state.region;
    if (box.areaIsZero(region)) return undefined;
    if (!this.state.visible) return () => renderCtx.erase(region, xy.ZERO, CANVAS);
    let range: Iterable<any>;
    if (!this.state.scrolling)
      range = this.values.subIterator(
        this.values.length - this.visibleLineCount,
        this.values.length,
      );
    else {
      const start = this.values.traverseAlignment(
        this.scrollState.offset,
        -BigInt(this.visibleLineCount),
      );
      range = this.values.subAlignmentSpanIterator(start, this.visibleLineCount);
    }

    const reg = this.state.region;
    const canvas = renderCtx[CANVAS];
    const draw2d = new Draw2D(canvas, this.internal.theme);
    const clearScissor = renderCtx.scissor(reg, xy.ZERO, [CANVAS]);
    this.renderElements(draw2d, range);
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
        { x: box.right(reg) - 6, y: scrollbarYPos },
        { width: 6, height: scrollbarHeight },
      ),
      bordered: false,
      backgroundColor: (t: theming.Theme) => t.colors.gray.l6,
    });
  }

  private renderElements(draw2D: Draw2D, iter: Iterable<TelemValue>): void {
    const reg = this.state.region;
    let i = 0;
    for (const value of iter) {
      const text = this.values.dataType.equals(DataType.JSON)
        ? JSON.stringify(value)
        : value.toString();
      draw2D.text({
        text,
        level: this.state.font,
        shade: 11,
        position: xy.translate(box.topLeft(reg), { x: 6, y: i * this.lineHeight + 6 }),
        code: true,
      });
      i++;
    }
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Log.TYPE]: Log };
