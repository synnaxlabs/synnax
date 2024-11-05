// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, scale, xy } from "@synnaxlabs/x/spatial";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { color } from "@/color/core";
import { telem } from "@/telem/aether";
import { text } from "@/text/core";
import { dimensions } from "@/text/dimensions";
import { theming } from "@/theming/aether";
import { type Element } from "@/vis/diagram/aether/Diagram";
import { render } from "@/vis/render";

const valueState = z.object({
  box: box.box,
  telem: telem.stringSourceSpecZ.optional().default(telem.noopStringSourceSpec),
  level: text.levelZ.optional().default("p"),
  color: color.Color.z.optional().default(color.ZERO),
  precision: z.number().optional().default(2),
  minWidth: z.number().optional().default(60),
  width: z.number().optional(),
});

const CANVAS_VARIANT: render.Canvas2DVariant = "upper2d";

export interface ValueProps {
  scale?: scale.XY;
}

interface InternalState {
  theme: theming.Theme;
  render: render.Context;
  telem: telem.StringSource;
  stopListening?: () => void;
  requestRender: render.RequestF | null;
  textColor: color.Color;
}

export class Value
  extends aether.Leaf<typeof valueState, InternalState>
  implements Element
{
  static readonly TYPE = "value";
  static readonly z = valueState;
  schema = Value.z;

  async afterUpdate(): Promise<void> {
    const { internal: i } = this;
    i.render = render.Context.use(this.ctx);
    i.theme = theming.use(this.ctx);
    if (this.state.color.isZero) this.internal.textColor = i.theme.colors.gray.l8;
    else i.textColor = this.state.color;
    i.telem = await telem.useSource(this.ctx, this.state.telem, i.telem);
    i.stopListening?.();
    i.stopListening = this.internal.telem.onChange(() => this.requestRender());
    this.internal.requestRender = render.Controller.useOptionalRequest(this.ctx);
    this.requestRender();
  }

  async afterDelete(): Promise<void> {
    const { requestRender, telem, render: renderCtx } = this.internal;
    await telem.cleanup?.();
    if (requestRender == null)
      renderCtx.erase(box.construct(this.state.box), xy.ZERO, CANVAS_VARIANT);
    else requestRender(render.REASON_LAYOUT);
  }

  private requestRender(): void {
    const { requestRender } = this.internal;
    if (requestRender != null) requestRender(render.REASON_LAYOUT);
    else void this.render({});
  }

  async render({ viewportScale = scale.XY.IDENTITY }): Promise<void> {
    const { render: renderCtx, telem, theme } = this.internal;
    const b = box.construct(this.state.box);
    if (box.areaIsZero(b)) return;
    const canvas = renderCtx[CANVAS_VARIANT].applyScale(viewportScale);
    const value = await telem.value();
    const fontString = theming.fontString(this.internal.theme, this.state.level);
    canvas.font = theming.fontString(this.internal.theme, this.state.level);
    const height = theme.typography[this.state.level].size * theme.sizes.base;
    const width = dimensions(value, fontString, canvas).width;
    if (this.internal.requestRender == null)
      renderCtx.erase(box.construct(this.prevState.box));
    const requiredWidth = width + theme.sizes.base;
    if (
      this.state.width == null ||
      this.state.width < requiredWidth ||
      (this.state.minWidth > requiredWidth && this.state.width !== this.state.minWidth)
    )
      this.setState((p) => ({ ...p, width: Math.max(requiredWidth, p.minWidth) }));
    const labelPosition = xy.couple(
      xy.translate(box.topLeft(b), { x: 12, y: (box.height(b) + height) / 2 }),
    );
    canvas.fillStyle = this.internal.textColor.hex;
    canvas.fillText(value, ...labelPosition);
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Value.TYPE]: Value,
};
