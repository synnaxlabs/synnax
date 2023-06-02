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

import { BobComponentFactory, BobComposite } from "@/core/bob/worker";
import { AxisCanvas } from "@/core/vis/Axis/AxisCanvas";
import { Axis, axisState } from "@/core/vis/Axis/core";
import { YAxis, YAxisContext, autoBounds } from "@/core/vis/LinePlot/worker/YAxis";
import { RenderContext } from "@/core/vis/render";

export const xAxisState = axisState.extend({
  location: Location.strictYZ.optional().default("bottom"),
  bound: Bound.looseZ.optional(),
  autoBoundPadding: z.number().optional().default(0.1),
});

export type XAxisState = z.input<typeof xAxisState>;
export type ParsedXAxisState = z.output<typeof xAxisState>;

export interface XAxisProps {
  plottingRegion: Box;
  viewport: Box;
}

export class XAxisFactory implements BobComponentFactory<XAxis> {
  ctx: RenderContext;
  yAxisFactory: BobComponentFactory<YAxis>;
  requestRender: () => void;

  constructor(
    ctx: RenderContext,
    yAxisFactory: BobComponentFactory<YAxis>,
    requestRender: () => void
  ) {
    this.ctx = ctx;
    this.yAxisFactory = yAxisFactory;
    this.requestRender = requestRender;
  }

  create(type: string, key: string, props: XAxisState): XAxis {
    return new XAxis(this.ctx, this.yAxisFactory, key, props, this.requestRender);
  }
}

export class XAxis extends BobComposite<YAxis, XAxisState, ParsedXAxisState> {
  ctx: RenderContext;
  core: Axis;
  static readonly TYPE = "x-axis";

  constructor(
    ctx: RenderContext,
    yAxisFactory: BobComponentFactory<YAxis>,
    key: string,
    props: XAxisState,
    requestRender: () => void
  ) {
    super(XAxis.TYPE, key, yAxisFactory, xAxisState, props);
    this.ctx = ctx;
    this.core = new AxisCanvas(ctx, this.state);
    this.setHook(() => {
      this.core.setState(this.state);
      requestRender();
    });
  }

  async render(props: XAxisProps): Promise<void> {
    const [normal, offset] = await this.scales(props);
    await this.renderAxis(props, normal);
    await this.renderYAxes(props, offset);
  }

  private async renderAxis(ctx: XAxisProps, scale: Scale): Promise<void> {
    this.core.render({ ...ctx, scale });
  }

  private async renderYAxes(ctx: XAxisProps, scale: Scale): Promise<void> {
    await Promise.all(
      this.children.map(async (el, i) => {
        const _ctx: YAxisContext = {
          plottingRegion: ctx.plottingRegion,
          viewport: ctx.viewport,
          xScale: scale,
        };
        await el.render(_ctx);
      })
    );
  }

  async xBound(): Promise<[Bound, number]> {
    if (this.state.bound != null) return [this.state.bound, this.state.bound.lower];
    const bounds = await Promise.all(
      this.children.map(async (el) => await el.xBound())
    );
    if (bounds.every((bound) => !bound.isFinite))
      return [new Bound({ lower: 0, upper: 1 }), 0];
    const { autoBoundPadding = 0.1 } = this.state;
    return autoBounds(autoBoundPadding, bounds);
  }

  private async scales(ctx: XAxisProps): Promise<[Scale, Scale]> {
    const [bound] = await this.xBound();
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
