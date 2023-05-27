import { Bound, Box, Scale, maxBound } from "@synnaxlabs/x";

import { TelemProvider } from "../telem";
import { WComponent, WorkerMessage } from "../worker/worker";

import { WLine, WLineContext, WLineProgram } from "@/core/vis/Line/WLine";

export interface YAxisProps {
  key: string;
  bound?: Bound;
  boundPadding: number;
  label: string;
  tickSpacing: number;
  type: "time" | "linear";
}

export interface YAxisContext {
  aspect: number;
  viewport: Box;
  xOffset: number;
  xScale: number;
}

export class WYAxis implements WComponent {
  props: YAxisProps;
  lines: WLine[];

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
    const lineCtx: WLineContext = {
      aspect: ctx.aspect,
      transform: {
        offset: {
          x: ctx.xOffset,
          y: yOffsetScale.pos(0),
        },
        scale: {
          x: ctx.xScale,
          y: yOffsetScale.dim(1),
        },
      },
    };
    await Promise.all(this.lines.map(async (el) => await el.render(lineCtx)));
  }

  private async yBound(): Promise<Bound> {
    if (this.props.bound != null) return this.props.bound;
    const { upper, lower } = maxBound(
      await Promise.all(this.lines.map(async (el) => await el.yBound()))
    );
    if (upper === lower) return { lower: lower - 1, upper: upper - 1 };
    const _padding = (upper - lower) * this.props.boundPadding;
    return { lower: lower - _padding, upper: upper + _padding };
  }

  private async yOffsetScale(ctx: YAxisContext): Promise<Scale> {
    const bound = await this.yBound();
    const mag = 1 / ctx.viewport.y;
    const trans = -ctx.viewport.x;
    return Scale.scale(bound).translate(-bound.lower).translate(trans).magnify(mag);
  }
}
