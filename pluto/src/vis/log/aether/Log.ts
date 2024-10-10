// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Destructor } from "@synnaxlabs/x";
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
  telem: telem.stringSourceSpecZ.optional().default(telem.noopStringSourceSpec),
  font: text.levelZ.optional().default("p"),
  color: color.Color.z.optional().default(color.ZERO),
});

class Values {
  entries: string[] = [];

  push(value: string): void {
    this.entries.push(value);
  }
}

interface InternalState {
  theme: theming.Theme;
  render: render.Context;
  telem: telem.StringSource;
  textColor: color.Color;
  stopListeningTelem?: Destructor;
}

export class Log extends aether.Leaf<typeof logState, InternalState> {
  static readonly TYPE = "log";
  static readonly z = logState;
  schema = Log.z;
  values: Values = new Values();
  offsetRef: number = 0;

  async afterUpdate(): Promise<void> {
    const { internal: i } = this;
    i.render = render.Context.use(this.ctx);
    i.theme = theming.use(this.ctx);
    if (this.state.color.isZero) this.internal.textColor = i.theme.colors.gray.l8;
    else i.textColor = this.state.color;
    i.telem = await telem.useSource(this.ctx, this.state.telem, i.telem);
    await this.internal.telem.value();
    if (this.state.scrollPosition != null && this.prevState.scrollPosition == null) {
      this.offsetRef = this.values.entries.length;
    }
    i.stopListeningTelem?.();
    i.stopListeningTelem = i.telem.onChange(() => {
      this.internal.telem.value().then((v) => {
        this.values.push(v);
        this.requestRender();
      });
    });
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
    const totalHeight = this.values.entries.length * this.lineHeight;
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
      this.values.entries.length,
    );
    const clearScissor = renderCtx.scissor(b, xy.ZERO, ["upper2d"]);
    let range: [number, number];
    if (this.state.scrollPosition == null)
      range = [
        this.values.entries.length - visibleLineCount,
        this.values.entries.length,
      ];
    else {
      // scrollPosition tells us how many pixels we've moved in relation
      // to the offset ref.
      const scrollPos = Math.ceil(this.state.scrollPosition / lineHeight);
      range = [
        this.offsetRef + scrollPos - visibleLineCount,
        this.offsetRef + scrollPos,
      ];
    }
    this.renderElements(range[0], range[1]);
    this.maybeUpdateTotalHeight();
    clearScissor();
    return async ({ canvases }) => renderCtx.erase(b, xy.ZERO, ...canvases);
  }

  private renderElements(start: number, end: number): void {
    const { render: renderCtx } = this.internal;
    const b = box.construct(this.state.region);
    if (box.areaIsZero(b)) return;
    const canvas = renderCtx.upper2d;
    const draw2d = new Draw2D(canvas, this.internal.theme);
    const lineHeight =
      this.internal.theme.typography[this.state.font].size *
      this.internal.theme.sizes.base;
    const clearScissor = renderCtx.scissor(b, xy.ZERO, ["upper2d"]);

    for (let i = start; i < end; i++) {
      const value = this.values.entries.at(i);
      if (value == null) continue;
      draw2d.text({
        text: `${i}-${value}`,
        level: this.state.font,
        shade: 9,
        position: xy.translateY(box.topLeft(b), (i - start) * lineHeight),
      });
    }
    clearScissor();
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Log.TYPE]: Log };
