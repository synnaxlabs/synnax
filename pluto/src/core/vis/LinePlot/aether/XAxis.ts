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

import { LookupResult } from "../../Line/core";

import { calculateAxisPosition, GridPositionMeta } from "./LinePlot";

import { AetherComposite } from "@/core/aether/worker";
import { CSS } from "@/core/css";
import { ThemeContext } from "@/core/theming/aether";
import { fontString } from "@/core/theming/fontString";
import { AxisCanvas } from "@/core/vis/Axis/AxisCanvas";
import { Axis, axisState } from "@/core/vis/Axis/core";
import { autoBounds, withinSizeThreshold } from "@/core/vis/LinePlot/aether/axis";
import { AetherYAxis } from "@/core/vis/LinePlot/aether/YAxis";
import { RenderContext, RenderController } from "@/core/vis/render";

const xAxisState = axisState
  .extend({
    location: Location.strictYZ.optional().default("bottom"),
    bound: Bounds.looseZ.optional(),
    autoBoundPadding: z.number().optional().default(0.01),
    size: z.number().optional().default(0),
    labelSize: z.number().optional().default(0),
  })
  .partial({
    color: true,
    font: true,
    gridColor: true,
  });

export interface XAxisProps {
  plottingRegion: Box;
  viewport: Box;
  region: Box;
  grid: GridPositionMeta[];
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
    const theme = ThemeContext.use(this.ctx);
    return {
      ctx: renderCtx,
      core: new AxisCanvas(renderCtx, {
        color: theme.colors.gray.p2,
        font: fontString(theme, "small"),
        gridColor: theme.colors.gray.m2,
        ...this.state,
        size: this.state.size + this.state.labelSize,
      }),
    };
  }

  afterUpdate(): void {
    RenderController.requestRender(this.ctx);
  }

  async render(props: XAxisProps): Promise<void> {
    const [reversed, normal] = await this.scales(props);
    await this.renderAxis(props, reversed);
    await this.renderYAxes(props, normal);
  }

  private async renderAxis(props: XAxisProps, scale: Scale): Promise<void> {
    const { core } = this.derived;
    const { size } = core.render({
      ...props,
      position: calculateAxisPosition(this.key, props.grid, props.plottingRegion),
      scale,
    });
    if (!withinSizeThreshold(this.state.size, size))
      this.setState((p) => ({ ...p, size }));
  }

  private async renderYAxes(props: XAxisProps, scale: Scale): Promise<void> {
    await Promise.all(
      this.children.map(
        async (el) =>
          await el.render({
            grid: props.grid,
            plottingRegion: props.plottingRegion,
            viewport: props.viewport,
            scale,
            region: props.region,
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
                grid: props.grid,
                plottingRegion: props.plottingRegion,
                viewport: props.viewport,
                scale: (await this.scales(props))[1],
                region: props.region,
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
