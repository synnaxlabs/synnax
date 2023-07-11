// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Bounds, Box, Direction, Location, Scale, XY } from "@synnaxlabs/x";
import { z } from "zod";

import { LineGL } from "../../Line/LineGL";
import { AetherRule } from "../../Rule/aether";

import { AetherComposite } from "@/core/aether/worker";
import { CSS } from "@/core/css";
import { Axis, AxisCanvas } from "@/core/vis/Axis";
import { axisState } from "@/core/vis/Axis/core";
import { LineComponent, LineProps, LookupResult } from "@/core/vis/Line/core";
import { autoBounds, withinSizeThreshold } from "@/core/vis/LinePlot/aether/axis";
import { RenderContext, RenderController } from "@/core/vis/render";

const yAxisState = axisState.extend({
  position: XY.z.optional(),
  location: Location.strictXZ.optional().default("left"),
  bounds: Bounds.looseZ.optional(),
  autoBoundPadding: z.number().optional().default(0.05),
  size: z.number().optional().default(0),
  labelSize: z.number().optional().default(0),
});

export interface YAxisProps {
  plottingRegion: Box;
  viewport: Box;
  region: Box;
  scale: Scale;
}

interface Derived {
  ctx: RenderContext;
  core: Axis;
}

export class AetherYAxis extends AetherComposite<
  typeof yAxisState,
  Derived,
  LineComponent | AetherRule
> {
  static readonly TYPE = CSS.BE("line-plot", "y-axis");
  static readonly z = yAxisState;
  schema = AetherYAxis.z;

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

  get lines(): LineComponent[] {
    return this.childrenOfType(LineGL.TYPE);
  }

  get rules(): AetherRule[] {
    return this.childrenOfType(AetherRule.TYPE);
  }

  async xBounds(): Promise<Bounds> {
    return Bounds.max(
      await Promise.all(this.lines.map(async (el) => await el.xBounds()))
    );
  }

  async render(props: YAxisProps): Promise<void> {
    if (this.state.position == null) return;
    const [normal, offset] = await this.scales(props);
    this.renderAxis(props, this.state.position, normal);
    await this.renderLines(props, offset);
    await this.renderRules(props, normal);
  }

  private renderAxis(props: YAxisProps, position: XY, scale: Scale): void {
    const { core } = this.derived;
    const { size } = core.render({ ...props, position, scale });
    if (!withinSizeThreshold(this.state.size, size))
      this.setState((p) => ({ ...p, size }));
  }

  private async renderLines(ctx: YAxisProps, scale: Scale): Promise<void> {
    const lineCtx: LineProps = {
      region: ctx.plottingRegion,
      scale: { x: ctx.scale, y: scale },
    };
    await Promise.all(this.lines.map(async (el) => el.render(lineCtx)));
  }

  private async renderRules(ctx: YAxisProps, scale: Scale): Promise<void> {
    const clearScissor = this.derived.ctx.scissorCanvas(ctx.plottingRegion);
    await Promise.all(
      this.rules.map(
        async (el) =>
          await el.render({
            ...ctx,
            scale,
            direction: Direction.x,
          })
      )
    );
    clearScissor();
  }

  private async yBounds(): Promise<[Bounds, number]> {
    if (this.state.bounds != null && !this.state.bounds.isZero)
      return [this.state.bounds, this.state.bounds.lower];
    const bounds = await Promise.all(this.lines.map(async (el) => await el.yBounds()));
    return autoBounds(bounds, this.state.autoBoundPadding, this.state.type);
  }

  async lookupX(props: YAxisProps, value: number): Promise<LookupResult[]> {
    const [normal, offset] = await this.scales(props);
    return await Promise.all(
      this.lines.map(
        async (el) =>
          await el.searchX(
            { region: props.plottingRegion, scale: { x: normal, y: offset } },
            value
          )
      )
    );
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
