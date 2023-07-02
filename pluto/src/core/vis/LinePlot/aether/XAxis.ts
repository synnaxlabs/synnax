// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Bounds, Box, Location, Scale } from "@synnaxlabs/x";
import { z } from "zod";

import { AetherComposite, AetherContext, AetherUpdate } from "@/core/aether/worker";
import { CSS } from "@/core/css";
import { AxisCanvas } from "@/core/vis/Axis/AxisCanvas";
import { Axis, axisState } from "@/core/vis/Axis/core";
import { autoBounds } from "@/core/vis/LinePlot/aether/axis";
import { YAxis } from "@/core/vis/LinePlot/aether/YAxis";
import { RenderContext, RenderController } from "@/core/vis/render";

const xAxisState = axisState.extend({
  location: Location.strictYZ.optional().default("bottom"),
  bound: Bounds.looseZ.optional(),
  autoBoundPadding: z.number().optional().default(0.01),
});

export interface XAxisProps {
  plottingRegion: Box;
  viewport: Box;
}

export class XAxis extends AetherComposite<typeof xAxisState, YAxis> {
  ctx: RenderContext;
  core: Axis;

  static readonly TYPE = CSS.BE("line-plot", "x-axis");
  static readonly stateZ = xAxisState;

  constructor(update: AetherUpdate) {
    super(update, xAxisState);
    this.ctx = RenderContext.use(update.ctx);
    this.core = new AxisCanvas(this.ctx, this.state);
  }

  handleUpdate(ctx: AetherContext): void {
    this.core.setState(this.state);
    RenderController.requestRender(ctx);
  }

  async render(props: XAxisProps): Promise<void> {
    const [reversed, normal] = await this.scales(props);
    await this.renderAxis(props, reversed);
    await this.renderYAxes(props, normal);
  }

  private async renderAxis(ctx: XAxisProps, scale: Scale): Promise<void> {
    this.core.render({ ...ctx, scale });
  }

  private async renderYAxes(ctx: XAxisProps, scale: Scale): Promise<void> {
    await Promise.all(
      this.children.map(
        async (el) =>
          await el.render({
            plottingRegion: ctx.plottingRegion,
            viewport: ctx.viewport,
            xScale: scale,
          })
      )
    );
  }

  async xBounds(): Promise<[Bounds, number]> {
    if (this.state.bound != null) return [this.state.bound, this.state.bound.lower];
    const bounds = await Promise.all(
      this.children.map(async (el) => await el.xBounds())
    );
    if (bounds.every((bound) => !bound.isFinite))
      return [new Bounds({ lower: 0, upper: 1 }), 0];
    return autoBounds(bounds, this.state.autoBoundPadding);
  }

  private async scales(ctx: XAxisProps): Promise<[Scale, Scale]> {
    const [bound] = await this.xBounds();
    return [
      Scale.scale(bound)
        .scale(1)
        .translate(-ctx.viewport.x)
        .magnify(1 / ctx.viewport.width)
        .reverse(),
      Scale.scale(bound)
        .scale(1)
        .translate(-ctx.viewport.x)
        .magnify(1 / ctx.viewport.width),
    ];
  }
}
