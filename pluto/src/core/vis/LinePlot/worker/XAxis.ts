import { Bound, Box, Scale, bound, yLocation } from "@synnaxlabs/x";
import { z } from "zod";

import { WComponentFactory, WComposite } from "@/core/bob/worker";
import { AxisCanvas } from "@/core/vis/Axis/AxisCanvas";
import { Axis, axisProps } from "@/core/vis/Axis/core";
import { YAxis, YAxisContext, autoBounds } from "@/core/vis/LinePlot/worker/YAxis";
import { RenderContext } from "@/core/vis/render";

export const xAxisState = axisProps.extend({
  location: yLocation.optional().default("bottom"),
  bound: bound.optional(),
  autoBoundPadding: z.number().optional().default(0.1),
});

export type XAxisState = z.input<typeof xAxisState>;
export type ParsedXAxisState = z.output<typeof xAxisState>;

export interface XAxisProps {
  plottingRegion: Box;
  viewport: Box;
}

export class XAxisFactory implements WComponentFactory<XAxis> {
  ctx: RenderContext;
  yAxisFactory: WComponentFactory<YAxis>;
  requestRender: () => void;

  constructor(
    ctx: RenderContext,
    yAxisFactory: WComponentFactory<YAxis>,
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

export class XAxis extends WComposite<YAxis, XAxisState, ParsedXAxisState> {
  ctx: RenderContext;
  core: Axis;
  static readonly TYPE = "x-axis";

  constructor(
    ctx: RenderContext,
    yAxisFactory: WComponentFactory<YAxis>,
    key: string,
    props: XAxisState,
    requestRender: () => void
  ) {
    super(XAxis.TYPE, key, yAxisFactory, xAxisState, props);
    this.ctx = ctx;
    this.core = new AxisCanvas(ctx, this.state);
    this.setHook(requestRender);
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
    if (bounds.every((bound) => !isFinite(bound.lower) || !isFinite(bound.upper)))
      return [
        {
          lower: 0,
          upper: 1,
        },
        0,
      ];
    const { autoBoundPadding = 0.1 } = this.state;
    return autoBounds(autoBoundPadding, bounds);
  }

  private async scales(ctx: XAxisProps): Promise<[Scale, Scale]> {
    const [bound, offset] = await this.xBound();
    return [
      Scale.scale(bound)
        .scale(1)
        .translate(ctx.viewport.x)
        .magnify(1 / ctx.viewport.width)
        .reverse(),
      Scale.scale(bound)
        .scale(1)
        .translate(-offset)
        .translate(ctx.viewport.x)
        .magnify(1 / ctx.viewport.width),
    ];
  }
}
