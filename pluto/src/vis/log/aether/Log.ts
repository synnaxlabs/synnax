// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Destructor, MultiSeries } from "@synnaxlabs/x";
import { bounds, box, xy } from "@synnaxlabs/x/spatial";
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
  scrollPosition: z.number(),
  scrollback: z.boolean(),
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
  overScrollCompensation: number;
}

export class Log extends aether.Leaf<typeof logState, InternalState> {
  static readonly TYPE = "log";
  static readonly z = logState;
  schema = Log.z;
  values: MultiSeries = new MultiSeries([]);
  scrollback: ScrollbackState | null = null;

  async afterUpdate(): Promise<void> {
    const { internal: i } = this;
    i.render = render.Context.use(this.ctx);
    i.theme = theming.use(this.ctx);
    if (this.state.color.isZero) this.internal.textColor = i.theme.colors.gray.l8;
    else i.textColor = this.state.color;
    i.telem = await telem.useSource(this.ctx, this.state.telem, i.telem);

    if (this.state.scrollback == false && this.prevState.scrollback == true)
      this.scrollback = null;

    // We're in scrollback mode
    if (this.scrollback != null) {
      const dist = Math.ceil(
        (this.state.scrollPosition - this.scrollback.scrollRef) / this.lineHeight,
      );
      this.scrollback.offset = this.scrollback.offsetRef - BigInt(dist);
      // This means we've scrolled to the very first element.
      if (
        this.scrollback.offset <
        this.values.alignmentBounds.lower + BigInt(this.visibleLineCount)
      ) {
        this.scrollback.offset = this.values.alignmentBounds.lower;
        this.setState((s) => ({ ...s, scrollPosition: this.prevState.scrollPosition }));
      }
      // Should we exit scrollback mode?
      if (this.scrollback.offset >= this.values.alignmentBounds.upper) {
        this.scrollback = null;
        this.setState((s) => ({ ...s, scrollback: false }));
      }
    } else {
      // Not in scrollback mode. Should we enter it?
      const scrolledUp = this.state.scrollPosition > this.prevState.scrollPosition;
      if (scrolledUp) {
        const off = this.values.alignmentBounds.upper - 1n;
        this.scrollback = {
          offset: off,
          offsetRef: off,
          scrollRef: this.state.scrollPosition,
          overScrollCompensation: 0,
        };
        this.setState((s) => ({ ...s, scrollback: true }));
      }
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
    const b = box.construct(this.state.region);
    if (box.areaIsZero(b)) return undefined;
    if (!this.state.visible) return async () => renderCtx.erase(b, xy.ZERO, "upper2d");
    const clearScissor = renderCtx.scissor(b, xy.ZERO, ["upper2d"]);
    let range: Iterable<any>;
    if (this.scrollback == null)
      range = this.values.subIterator(
        this.values.length - this.visibleLineCount,
        this.values.length,
      );
    else {
      range = this.values.subAlignmentSpanIterator(
        this.scrollback.offset - BigInt(this.visibleLineCount),
        this.visibleLineCount,
      );
    }

    this.renderElements(range);
    clearScissor();
    return async ({ canvases }) => renderCtx.erase(b, xy.ZERO, ...canvases);
  }

  private renderElements(iter: Iterable<any>): void {
    const { render: renderCtx } = this.internal;
    const reg = this.state.region;
    if (box.areaIsZero(reg)) return;
    const canvas = renderCtx.upper2d;
    const draw2d = new Draw2D(canvas, this.internal.theme);
    const clearScissor = renderCtx.scissor(reg, xy.ZERO, ["upper2d"]);
    let i = 0;
    const bHeight = (box.height(reg) / this.totalHeight) * box.height(reg);
    let yPos = box.bottom(reg) - bHeight;
    if (this.scrollback != null)
      yPos -=
        (Number(this.values.alignmentBounds.upper - this.scrollback.offset) /
          Number(bounds.span(this.values.alignmentBounds))) *
        box.height(reg);

    if (yPos < 0) yPos = box.top(reg);

    draw2d.container({
      region: box.construct(
        {
          x: box.right(reg) - 6,
          y: yPos,
        },
        { width: 6, height: bHeight },
      ),
      bordered: false,
      backgroundColor: (t) => t.colors.gray.l4,
    });
    for (const value of iter) {
      draw2d.text({
        text: this.values.dataType.equals(DataType.JSON)
          ? JSON.stringify(value)
          : value.toString(),
        level: this.state.font,
        shade: 9,
        position: xy.translate(box.topLeft(reg), { x: 6, y: i * this.lineHeight + 6 }),
        code: true,
      });
      i++;
    }
    clearScissor();
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Log.TYPE]: Log };
