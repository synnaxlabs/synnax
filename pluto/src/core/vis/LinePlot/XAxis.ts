import { Bound, Box, Scale, XY } from "@synnaxlabs/x";

import { Axis, AxisCanvas, AxisContext, AxisProps } from "../Axis";
import { RenderContext } from "../render";

import { YAxis, YAxisContext, YAxisProps, autoBounds } from "@/core/vis/LinePlot/YAxis";

export interface XAxisProps extends AxisProps {
  key: string;
  bound?: Bound;
  autoBoundPadding?: number;
}

export interface XAxisContext extends Omit<AxisContext, "scale"> {
  region: Box;
  viewport: Box;
  yAxisPositions: XY[];
  yAxisSizes: number[];
}

export class XAxis {
  ctx: RenderContext;
  props: XAxisProps;
  axes: YAxis[];
  core: Axis;
  static readonly TYPE = "x-axis";

  constructor(ctx: RenderContext, props: XAxisProps) {
    this.ctx = ctx;
    this.props = props;
    this.axes = [];
    this.core = new AxisCanvas(ctx, props);
  }

  setProps(props: YAxisProps): void {
    this.props = props;
    this.axes = [];
    this.core.setProps(props);
  }

  get key(): string {
    return this.props.key;
  }

  async render(ctx: XAxisContext): Promise<void> {
    const [normal, offset] = await this.scales(ctx);
    await this.renderAxis(ctx, normal);
    await this.renderYAxes(ctx, offset);
  }

  private async renderAxis(ctx: XAxisContext, scale: Scale): Promise<void> {
    this.core.render({ ...ctx, scale });
  }

  private async renderYAxes(ctx: XAxisContext, scale: Scale): Promise<void> {
    await Promise.all(
      this.axes.map(async (el, i) => {
        const _ctx: YAxisContext = {
          region: ctx.region,
          viewport: ctx.viewport,
          position: ctx.yAxisPositions[i],
          size: ctx.yAxisSizes[i],
          xScale: scale,
        };
        await el.render(_ctx);
      })
    );
  }

  async xBound(): Promise<[Bound, number]> {
    if (this.props.bound != null) return [this.props.bound, this.props.bound.lower];
    const bounds = await Promise.all(this.axes.map(async (el) => await el.xBound()));
    const { autoBoundPadding = 0.1 } = this.props;
    return autoBounds(autoBoundPadding, bounds);
  }

  private async scales(ctx: XAxisContext): Promise<[Scale, Scale]> {
    const [bound, offset] = await this.xBound();
    return [
      Scale.scale(bound)
        .scale(1)
        .translate(ctx.viewport.x)
        .magnify(1 / ctx.viewport.width),
      Scale.scale(bound)
        .scale(1)
        .translate(-offset)
        .translate(ctx.viewport.x)
        .magnify(1 / ctx.viewport.width),
    ];
  }
}
