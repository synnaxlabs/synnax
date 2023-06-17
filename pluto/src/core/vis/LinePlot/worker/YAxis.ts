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

import { autoBounds } from "./axis";

import { AetherComposite, Update } from "@/core/aether/worker";
import { Axis, AxisCanvas } from "@/core/vis/Axis";
import { axisState } from "@/core/vis/Axis/core";
import { LineComponent, LineProps } from "@/core/vis/Line/core";
import { RenderContext, RenderController } from "@/core/vis/render";

export const yAxisState = axisState.extend({
  location: Location.strictXZ.optional().default("left"),
  bound: Bounds.looseZ.optional(),
  autoBoundPadding: z.number().optional().default(0.1),
});

export type YAxisState = z.input<typeof yAxisState>;

export interface YAxisProps {
  plottingRegion: Box;
  viewport: Box;
  xScale: Scale;
}

export class YAxis extends AetherComposite<typeof yAxisState, LineComponent> {
  ctx: RenderContext;
  core: Axis;

  static readonly TYPE = "y-axis";

  constructor(update: Update) {
    super(update, yAxisState);
    this.ctx = RenderContext.use(update.ctx);
    this.core = new AxisCanvas(this.ctx, this.state);
    this.onUpdate((ctx) => {
      this.core.setState(this.state);
      RenderController.requestRender(ctx);
    });
  }

  async xBounds(): Promise<Bounds> {
    return Bounds.max(
      await Promise.all(this.children.map(async (el) => await el.xBounds()))
    );
  }

  async render(ctx: YAxisProps): Promise<void> {
    const [normal, offset] = await this.scales(ctx);
    this.renderAxis(ctx, normal);
    await this.renderLines(ctx, offset);
  }

  private renderAxis(ctx: YAxisProps, scale: Scale): void {
    this.core.render({ ...ctx, scale });
  }

  private async renderLines(ctx: YAxisProps, scale: Scale): Promise<void> {
    const lineCtx: LineProps = {
      region: ctx.plottingRegion,
      scale: { x: ctx.xScale, y: scale },
    };
    await Promise.all(this.children.map(async (el) => el.render(lineCtx)));
  }

  private async yBounds(): Promise<[Bounds, number]> {
    if (this.state.bound != null) return [this.state.bound, this.state.bound.lower];
    const bounds = await Promise.all(
      this.children.map(async (el) => await el.yBounds())
    );
    return autoBounds(bounds, this.state.autoBoundPadding);
  }

  private async scales(ctx: YAxisProps): Promise<[Scale, Scale]> {
    const [bound] = await this.yBounds();
    return [
      Scale.scale(bound)
        .scale(1)
        .translate(-ctx.viewport.y)
        .magnify(1 / ctx.viewport.height)
        .invert()
        .reverse(),
      Scale.scale(bound)
        .scale(1)
        .translate(-ctx.viewport.y)
        .magnify(1 / ctx.viewport.height),
    ];
  }
}
