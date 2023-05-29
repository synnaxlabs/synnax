import { Bound, Box, Scale, XY, maxBound } from "@synnaxlabs/x";

import { LineContext } from "../Line/core";
import { RenderContext } from "../render";

import { Axis, AxisCanvas, AxisProps } from "@/core/vis/Axis";
import { LineRenderer } from "@/core/vis/Line";

export interface YAxisProps extends AxisProps {
  key: string;
  bound?: Bound;
  autoBoundPadding?: number;
}

export interface YAxisContext {
  region: Box;
  viewport: Box;
  xScale: Scale;
  position: XY;
  size: number;
}

export class YAxis {
  ctx: RenderContext;
  props: YAxisProps;
  lines: LineRenderer[];
  core: Axis;

  static readonly TYPE = "y-axis";

  constructor(ctx: RenderContext, props: YAxisProps) {
    this.ctx = ctx;
    this.props = props;
    this.lines = [];
    this.core = new AxisCanvas(ctx, props);
  }

  get key(): string {
    return this.props.key;
  }

  setProps(props: YAxisProps): void {
    this.props = props;
    this.core.setProps(props);
  }

  async xBound(): Promise<Bound> {
    return maxBound(await Promise.all(this.lines.map(async (el) => await el.xBound())));
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
      region: ctx.region,
      scale: { x: ctx.xScale, y: scale },
    };
    await Promise.all(this.lines.map(async (el) => el.render(lineCtx)));
  }

  private async yBound(): Promise<[Bound, number]> {
    if (this.props.bound != null) return [this.props.bound, this.props.bound.lower];
    const bounds = await Promise.all(this.lines.map(async (el) => await el.yBound()));
    const { autoBoundPadding = 0.1 } = this.props;
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
  const { upper, lower } = maxBound(bounds);
  if (upper === lower) return [{ lower: lower - 1, upper: upper - 1 }, lower];
  const _padding = (upper - lower) * padding;
  return [{ lower: lower - _padding, upper: upper + _padding }, lower];
};
