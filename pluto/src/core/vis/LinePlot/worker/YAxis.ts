import { Bound, Box, Scale, bound, maxBound, xLocation } from "@synnaxlabs/x";
import { z } from "zod";

import { WComposite, WComponentFactory } from "@/core/bob/worker";
import { Axis, AxisCanvas } from "@/core/vis/Axis";
import { axisProps } from "@/core/vis/Axis/core";
import { LineComponent, LineContext } from "@/core/vis/Line/core";
import { RenderContext } from "@/core/vis/render";

export const yAxisProps = axisProps.extend({
  location: xLocation.optional().default("left"),
  bound: bound.optional(),
  autoBoundPadding: z.number().optional().default(0.1),
});

export type YAxisState = z.input<typeof yAxisProps>;
export type ParsedYAxisState = z.output<typeof yAxisProps>;

export interface YAxisContext {
  plottingRegion: Box;
  viewport: Box;
  xScale: Scale;
}

export class YAxisFactory implements WComponentFactory<YAxis> {
  ctx: RenderContext;
  lineFactory: WComponentFactory<LineComponent>;
  requestRender: () => void;

  constructor(
    ctx: RenderContext,
    lineFactory: WComponentFactory<LineComponent>,
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

export class YAxis extends WComposite<LineComponent, YAxisState, ParsedYAxisState> {
  ctx: RenderContext;
  core: Axis;

  static readonly TYPE = "y-axis";

  constructor(
    ctx: RenderContext,
    lineFactory: WComponentFactory<LineComponent>,
    key: string,
    props: YAxisState,
    requestRender: () => void
  ) {
    super(YAxis.TYPE, key, lineFactory, yAxisProps, props);
    this.ctx = ctx;
    this.core = new AxisCanvas(ctx, this.state);
    this.setHook(requestRender);
  }

  async xBound(): Promise<Bound> {
    return maxBound(
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
    const [bound, offset] = await this.yBound();
    return [
      Scale.scale(bound)
        .scale(1)
        .translate(-ctx.viewport.y)
        .magnify(1 / ctx.viewport.height),
      Scale.scale(bound)
        .translate(-offset)
        .scale(1)
        .translate(-ctx.viewport.y)
        .magnify(1 / ctx.viewport.height),
    ];
  }
}

export const autoBounds = (padding: number, bounds: Bound[]): [Bound, number] => {
  if (bounds.length === 0) return [{ lower: 0, upper: 1 }, 0];
  const { upper, lower } = maxBound(bounds);
  if (upper === lower) return [{ lower: lower - 1, upper: upper - 1 }, lower];
  const _padding = (upper - lower) * padding;
  return [{ lower: lower - _padding, upper: upper + _padding }, lower];
};
