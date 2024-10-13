// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Destructor, MultiSeries, TelemValue } from "@synnaxlabs/x";
import { box, xy } from "@synnaxlabs/x/spatial";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { color } from "@/color/core";
import { telem } from "@/telem/aether";
import { text } from "@/text/core";
import { theming } from "@/theming/aether";
import { Draw2D } from "@/vis/draw2d";
import { render } from "@/vis/render";

export const logState = z.object({
  region: box.box,
  wheelPos: z.number(),
  scrolling: z.boolean(),
  visible: z.boolean(),
  telem: telem.seriesSourceSpecZ.optional().default(telem.noopSeriesSourceSpec),
  font: text.levelZ.optional().default("p"),
  color: color.Color.z.optional().default(color.ZERO),
});

interface InternalState {
  theme: theming.Theme;
  render: render.Context;
  telem: telem.SeriesSource;
  textColor: color.Color;
  stopListeningTelem?: Destructor;
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

  async afterUpdate(): Promise<void> {
    const { internal: i } = this;
    i.render = render.Context.use(this.ctx);
    i.theme = theming.use(this.ctx);
    if (this.state.color.isZero) this.internal.textColor = i.theme.colors.gray.l8;
    else i.textColor = this.state.color;
    i.telem = await telem.useSource(this.ctx, this.state.telem, i.telem);

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
      // console.log({
      //   bounds: this.values.series.map((s) => s.alignmentBounds),
      //   start: scrollState.offsetRef,
      //   dist: -BigInt(dist),
      // });
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

    const [_, series] = await this.internal.telem.value();
    this.values = new MultiSeries(series);
    i.stopListeningTelem?.();
    i.stopListeningTelem = i.telem.onChange(() =>
      this.internal.telem.value().then(([_, series]) => {
        this.values = new MultiSeries(series);
        this.requestRender();
      }),
    );
    this.requestRender();
  }

  async afterDelete(): Promise<void> {
    const { telem, render: renderCtx } = this.internal;
    await telem.cleanup?.();
    renderCtx.erase(box.construct(this.state.region), xy.ZERO, "upper2d");
  }

  private requestRender(): void {
    const { render } = this.internal;
    render.loop.set({
      key: `${this.type}-${this.key}`,
      render: async () => await this.render(),
      priority: "high",
      canvases: ["lower2d", "upper2d"],
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

  async render(): Promise<render.Cleanup | undefined> {
    const { render: renderCtx } = this.internal;
    const region = this.state.region;
    if (box.areaIsZero(region)) return undefined;
    if (!this.state.visible)
      return async () => renderCtx.erase(region, xy.ZERO, "upper2d");
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
    const canvas = renderCtx.upper2d;
    const draw2d = new Draw2D(canvas, this.internal.theme);
    const clearScissor = renderCtx.scissor(reg, xy.ZERO, ["upper2d"]);
    this.renderElements(draw2d, range);
    this.renderScrollbar(draw2d);
    clearScissor();
    const eraseRegion = box.copy(this.state.region);
    return async ({ canvases }) =>
      renderCtx.erase(eraseRegion, { x: 10, y: 10 }, ...canvases);
  }

  private renderScrollbar(draw2d: Draw2D): void {
    const reg = this.state.region;
    const bHeight = (box.height(reg) / this.totalHeight) * box.height(reg);
    let yPos = box.bottom(reg) - bHeight;
    if (this.state.scrolling)
      yPos -=
        (Number(
          this.values.distance(
            this.values.alignmentBounds.upper,
            this.scrollState.offset,
          ),
        ) /
          this.values.length) *
        box.height(reg);

    if (yPos < 0) yPos = box.top(reg);

    draw2d.container({
      region: box.construct(
        { x: box.right(reg) - 6, y: yPos },
        { width: 6, height: bHeight },
      ),
      bordered: false,
      backgroundColor: (t) => t.colors.gray.l4,
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
        shade: 9,
        position: xy.translate(box.topLeft(reg), { x: 6, y: i * this.lineHeight + 6 }),
        code: true,
      });
      i++;
    }
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Log.TYPE]: Log };
