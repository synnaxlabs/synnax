// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Bound, Box, Location, Scale } from "@synnaxlabs/x";
import { z } from "zod";

import { BobComposite, BobComponentFactory } from "@/core/bob/worker";
import { Axis, AxisCanvas } from "@/core/vis/Axis";
import { axisState } from "@/core/vis/Axis/core";
import { LineComponent, LineContext } from "@/core/vis/Line/core";
import { RenderContext } from "@/core/vis/render";

export const yAxisProps = axisState.extend({
  location: Location.strictXZ.optional().default("left"),
  bound: Bound.looseZ.optional(),
  autoBoundPadding: z.number().optional().default(0.1),
});

export type YAxisState = z.input<typeof yAxisProps>;
export type ParsedYAxisState = z.output<typeof yAxisProps>;

export interface YAxisContext {
  plottingRegion: Box;
  viewport: Box;
  xScale: Scale;
}

export class YAxisFactory implements BobComponentFactory<YAxis> {
  ctx: RenderContext;
  lineFactory: BobComponentFactory<LineComponent>;
  requestRender: () => void;

  constructor(
    ctx: RenderContext,
    lineFactory: BobComponentFactory<LineComponent>,
    requestRender: () => void
  ) {
    this.ctx = ctx;
    this.lineFactory = lineFactory;
    this.requestRender = requestRender;
  }

  create(type: string, key: string, props: YAxisState): YAxis {
    return new YAxis(this.ctx, this.lineFactory, key, props, this.requestRender);
  }
}

export class YAxis extends BobComposite<LineComponent, YAxisState, ParsedYAxisState> {
  ctx: RenderContext;
  core: Axis;

  static readonly TYPE = "y-axis";

  constructor(
    ctx: RenderContext,
    lineFactory: BobComponentFactory<LineComponent>,
    key: string,
    props: YAxisState,
    requestRender: () => void
  ) {
    super(YAxis.TYPE, key, lineFactory, yAxisProps, props);
    this.ctx = ctx;
    this.core = new AxisCanvas(ctx, this.state);
    this.setHook(() => {
      this.core.setState(this.state);
      requestRender();
    });
  }

  async xBound(): Promise<Bound> {
    return Bound.max(
      await Promise.all(this.children.map(async (el) => await el.xBound()))
    );
  }

  async render(ctx: YAxisContext): Promise<void> {
    const [normal, offset] = await this.scales(ctx);
    this.renderAxis(ctx, normal);
    await this.renderLines(ctx, offset);
  }

  private renderAxis(ctx: YAxisContext, scale: Scale): void {
    this.core.render({ ...ctx, scale });
  }

  private async renderLines(ctx: YAxisContext, scale: Scale): Promise<void> {
    const lineCtx: LineContext = {
      region: ctx.plottingRegion,
      scale: { x: ctx.xScale, y: scale },
    };
    await Promise.all(this.children.map(async (el) => el.render(lineCtx)));
  }

  private async yBound(): Promise<[Bound, number]> {
    if (this.state.bound != null) return [this.state.bound, this.state.bound.lower];
    const bounds = await Promise.all(
      this.children.map(async (el) => await el.yBound())
    );
    const { autoBoundPadding = 0.1 } = this.state;
    return autoBounds(autoBoundPadding, bounds);
  }

  private async scales(ctx: YAxisContext): Promise<[Scale, Scale]> {
    const [bound] = await this.yBound();
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

export const autoBounds = (padding: number, bounds: Bound[]): [Bound, number] => {
  if (bounds.length === 0) return [new Bound({ lower: 0, upper: 1 }), 0];
  const { upper, lower } = Bound.max(bounds);
  if (upper === lower)
    return [new Bound({ lower: lower - 1, upper: upper - 1 }), lower];
  const _padding = (upper - lower) * padding;
  return [new Bound({ lower: lower - _padding, upper: upper + _padding }), lower];
};
