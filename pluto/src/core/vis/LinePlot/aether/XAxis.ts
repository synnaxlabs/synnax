// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Bounds, Box, Location, Scale, XY } from "@synnaxlabs/x";
import { z } from "zod";

import { LookupResult } from "../../Line/core";

import { AetherComposite } from "@/core/aether/worker";
import { CSS } from "@/core/css";
import { AxisCanvas } from "@/core/vis/Axis/AxisCanvas";
import { Axis, axisState } from "@/core/vis/Axis/core";
import { autoBounds, withinSizeThreshold } from "@/core/vis/LinePlot/aether/axis";
import { AetherYAxis } from "@/core/vis/LinePlot/aether/YAxis";
import { RenderContext, RenderController } from "@/core/vis/render";

const xAxisState = axisState.extend({
  location: Location.strictYZ.optional().default("bottom"),
  bound: Bounds.looseZ.optional(),
  autoBoundPadding: z.number().optional().default(0.01),
  size: z.number().optional().default(0),
  position: XY.z.optional(),
  labelSize: z.number().optional().default(0),
});

export interface XAxisProps {
  plottingRegion: Box;
  viewport: Box;
}

interface Derived {
  ctx: RenderContext;
  core: Axis;
}

export class AetherXAxis extends AetherComposite<
  typeof xAxisState,
  Derived,
  AetherYAxis
> {
  static readonly TYPE = CSS.BE("line-plot", "x-axis");
  static readonly z = xAxisState;
  schema = AetherXAxis.z;

  derive(): Derived {
    const renderCtx = RenderContext.use(this.ctx);
    return {
      ctx: renderCtx,
      core: new AxisCanvas(renderCtx, {
        ...this.state,
        size: this.state.size + this.state.labelSize,
      }),
    };
  }

  afterUpdate(): void {
    RenderController.requestRender(this.ctx);
  }

  async render(props: XAxisProps): Promise<void> {
    if (this.state.position == null) return;
    const [reversed, normal] = await this.scales(props);
    await this.renderAxis(props, this.state.position, reversed);
    await this.renderYAxes(props, normal);
  }

  private async renderAxis(ctx: XAxisProps, position: XY, scale: Scale): Promise<void> {
    const { core } = this.derived;
    const { size } = core.render({ ...ctx, position, scale });
    if (!withinSizeThreshold(this.state.size, size))
      this.setState((p) => ({ ...p, size }));
  }

  private async renderYAxes(ctx: XAxisProps, scale: Scale): Promise<void> {
    await Promise.all(
      this.children.map(
        async (el) =>
          await el.render({
            plottingRegion: ctx.plottingRegion,
            viewport: ctx.viewport,
            scale,
          })
      )
    );
  }

  async xBounds(): Promise<[Bounds, number]> {
    if (this.state.bound != null && !this.state.bound.isZero)
      return [this.state.bound, this.state.bound.lower];
    const bounds = (
      await Promise.all(this.children.map(async (el) => await el.xBounds()))
    ).filter((b) => b.isFinite);
    return autoBounds(bounds, this.state.autoBoundPadding, this.state.type);
  }

  async lookupX(props: XAxisProps, xValue: number): Promise<LookupResult[]> {
    return (
      await Promise.all(
        this.children.flatMap(
          async (el) =>
            await el.lookupX(
              {
                plottingRegion: props.plottingRegion,
                viewport: props.viewport,
                scale: (await this.scales(props))[1],
              },
              xValue
            )
        )
      )
    ).flat();
  }

  private async scales(ctx: XAxisProps): Promise<[Scale, Scale]> {
    const [bounds] = await this.xBounds();
    return [
      Scale.scale(bounds)
        .scale(1)
        .translate(-ctx.viewport.x)
        .magnify(1 / ctx.viewport.width)
        .reverse(),
      Scale.scale(bounds)
        .scale(1)
        .translate(-ctx.viewport.x)
        .magnify(1 / ctx.viewport.width),
    ];
  }
}

export interface XAxisLookupResult {
  position: number;
  yResults: LookupResult[];
}
