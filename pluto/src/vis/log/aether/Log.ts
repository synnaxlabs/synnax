// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
  telem: telem.stringSourceSpecZ.optional().default(telem.noopStringSourceSpec),
  font: text.levelZ.optional().default("p"),
  color: color.Color.z.optional().default(color.ZERO),
  precision: z.number().optional().default(2),
  minWidth: z.number().optional().default(60),
  width: z.number().optional(),
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
}

export class Log extends aether.Leaf<typeof logState, InternalState> {
  static readonly TYPE = "log";
  static readonly z = logState;
  schema = Log.z;
  values: Values = new Values();

  async afterUpdate(): Promise<void> {
    const { internal: i } = this;
    i.render = render.Context.use(this.ctx);
    i.theme = theming.use(this.ctx);
    if (this.state.color.isZero) this.internal.textColor = i.theme.colors.gray.l8;
    else i.textColor = this.state.color;
    i.telem = await telem.useSource(this.ctx, this.state.telem, i.telem);
    this.internal.telem.value();
    i.telem.onChange(() => {
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

  async render(): Promise<render.Cleanup | undefined> {
    const { render: renderCtx } = this.internal;
    const b = box.construct(this.state.region);
    if (box.areaIsZero(b)) return undefined;
    const canvas = renderCtx.upper2d;
    const draw2d = new Draw2D(canvas, this.internal.theme);
    const lineHeight =
      this.internal.theme.typography[this.state.font].size *
      this.internal.theme.sizes.base;

    const visibleLines = Math.floor(box.height(b) / lineHeight);

    for (let i = 0; i < visibleLines; i++) {
      const value = this.values.entries.at(-1 * (visibleLines - i)) as string;
      draw2d.text({
        text: value,
        level: this.state.font,
        shade: 5,
        position: xy.translateY(box.topLeft(b), i * lineHeight),
      });
    }
    return async ({ canvases }) => renderCtx.erase(b, xy.ZERO, ...canvases);
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Log.TYPE]: Log,
};
