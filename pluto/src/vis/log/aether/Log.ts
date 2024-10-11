// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Destructor, MultiSeries } from "@synnaxlabs/x";
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
  scrollPosition: z.number().or(z.null()),
  totalHeight: z.number(),
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

export class Log extends aether.Leaf<typeof logState, InternalState> {
  static readonly TYPE = "log";
  static readonly z = logState;
  schema = Log.z;
  values: MultiSeries = new MultiSeries([]);
  offsetRef: bigint = 0n;

  async afterUpdate(): Promise<void> {
    const { internal: i } = this;
    i.render = render.Context.use(this.ctx);
    i.theme = theming.use(this.ctx);
    if (this.state.color.isZero) this.internal.textColor = i.theme.colors.gray.l8;
    else i.textColor = this.state.color;
    i.telem = await telem.useSource(this.ctx, this.state.telem, i.telem);
    const [_, series] = await this.internal.telem.value();
    this.values = new MultiSeries(series);
    if (this.state.scrollPosition != null && this.prevState.scrollPosition == null)
      this.offsetRef = this.values.alignmentBounds.upper;
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

  private maybeUpdateTotalHeight(): void {
    // For the user, it really only matters if the size of the scroll bar
    // changes enough to be visually noticeable. We'll say 5px is the threshold.
    // 1. Calculate the total height
    const totalHeight = this.values.length * this.lineHeight;
    const prevScrollHeight = this.calculateScrollbarHeight(this.state.totalHeight);
    const nextScrollHeight = this.calculateScrollbarHeight(totalHeight);
    if (Math.abs(prevScrollHeight - nextScrollHeight) > 5)
      this.setState((p) => ({ ...p, totalHeight }));
  }

  private calculateScrollbarHeight(totalHeight: number): number {
    if (totalHeight < box.height(this.state.region)) return 0;
    return (
      (box.height(this.state.region) / totalHeight) * box.height(this.state.region)
    );
  }

  async render(): Promise<render.Cleanup | undefined> {
    const { render: renderCtx } = this.internal;
    const b = box.construct(this.state.region);
    if (box.areaIsZero(b)) return undefined;
    const lineHeight =
      this.internal.theme.typography[this.state.font].size *
      this.internal.theme.sizes.base;
    const visibleLineCount = Math.min(
      Math.floor(box.height(b) / lineHeight),
      this.values.length,
    );
    const clearScissor = renderCtx.scissor(b, xy.ZERO, ["upper2d"]);
    let range: Iterable<any>;
    if (this.state.scrollPosition == null)
      range = this.values.subIterator(
        this.values.length - visibleLineCount,
        this.values.length,
      );
    else {
      // scrollPosition tells us how many pixels we've moved in relation
      // to the offset ref.
      const scrollPos = BigInt(Math.ceil(this.state.scrollPosition / lineHeight));
      range = this.values.subAlignmentIterator(
        this.offsetRef + scrollPos - BigInt(visibleLineCount),
        this.offsetRef + scrollPos,
      );
    }
    this.renderElements(range);
    this.maybeUpdateTotalHeight();
    clearScissor();
    return async ({ canvases }) => renderCtx.erase(b, xy.ZERO, ...canvases);
  }

  private renderElements(iter: Iterable<any>): void {
    const { render: renderCtx } = this.internal;
    const b = box.construct(this.state.region);
    if (box.areaIsZero(b)) return;
    const canvas = renderCtx.upper2d;
    const draw2d = new Draw2D(canvas, this.internal.theme);
    const clearScissor = renderCtx.scissor(b, xy.ZERO, ["upper2d"]);
    let i = 0;
    for (const value of iter) {
      if (i % 100 == 0) console.log(i);
      draw2d.text({
        text: this.values.dataType.equals(DataType.JSON)
          ? JSON.stringify(value)
          : value.toString(),
        level: this.state.font,
        shade: 9,
        position: xy.translateY(box.topLeft(b), i * this.lineHeight),
        code: true,
      });
      i++;
    }
    clearScissor();
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Log.TYPE]: Log };
