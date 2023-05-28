import { Bound, Box, Scale, XY, maxBound, } from "@synnaxlabs/x";

import { WComponent } from "../worker/worker";

import { GLLine, GLLienContext } from "@/core/vis/Line/WLine";

export interface YAxisProps {
  key: string;
  boundPadding: number;
  label: string;
  tickSpacing: number;
  type: "time" | "linear";
  bound?: Bound;
}

export interface YAxisContext {
  region: Box;
  viewport: Box;
  xScale: Scale;
  position: XY;
}

export class WYAxis implements WComponent {
  props: YAxisProps;
  lines: GLLine[];

  static readonly TYPE = "y-axis";

  constructor(initialProps: YAxisProps) {
    this.props = initialProps;
    this.lines = [];
  }

  get key(): string {
    return this.props.key;
  }

  setProps(props: YAxisProps): void {
    this.props = props;
  }

  async xBound(): Promise<Bound> {
    return maxBound(await Promise.all(this.lines.map(async (el) => await el.xBound())));
  }

  async render(ctx: YAxisContext): Promise<void> {
    await this.renderAxis(ctx);
    await this.renderLines(ctx);
  }

  private async renderAxis(ctx: YAxisContext): Promise<void> {}

  private async renderLines(ctx: YAxisContext): Promise<void> {
    const yOffsetScale = await this.yOffsetScale(ctx);
    const lineCtx: GLLienContext = {
      region: ctx.region,
      scale: { x: ctx.xScale, y: yOffsetScale }
    };
    await Promise.all(this.lines.map(async (el) => await el.render(lineCtx)));
  }

  private async yBound(): Promise<[Bound, number]> {
    if (this.props.bound != null) return [this.props.bound, this.props.bound.lower];
    const { upper, lower } = maxBound(
      await Promise.all(this.lines.map(async (el) => await el.yBound()))
    );
    if (upper === lower) return [{ lower: lower - 1, upper: upper - 1 }, lower];
    const _padding = (upper - lower) * this.props.boundPadding;
    return [{ lower: lower - _padding, upper: upper + _padding }, lower];
  }

  private async yOffsetScale(ctx: YAxisContext): Promise<Scale> {
    const [offset, bound] = await this.yBound();
    return Scale.scale(bound).translate(-offset).translate(-ctx.viewport.y).magnify(1 / ctx.viewport.height);
  }
}
