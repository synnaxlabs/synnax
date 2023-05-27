import { Bound, Box } from "@synnaxlabs/x";

import { WLineProgram } from "../Line/WLine";
import { TelemProvider } from "../telem";
import { WComponent, WorkerMessage } from "../worker/worker";

import { WYAxis } from "@/core/vis/LinePlot/YAxis";

export interface XAxisProps {
  key: string;
  bound?: Bound;
  boundPadding: number;
  tickSpacing: number;
  type: "time" | "linear";
}

export interface WXAxisContext {
  viewport: Box;
  aspect: number;
}

export class WXAxis implements WComponent {
  props: XAxisProps;
  axes: WYAxis[];

  static readonly TYPE = "x-axis";

  constructor(props: XAxisProps) {
    this.props = props;
    this.axes = [];
  }

  setProps(props: XAxisProps): void {
    this.props = props;
    this.axes = [];
  }

  get key(): string {
    return this.props.key;
  }

  async render(ctx: WXAxisContext): Promise<void> {
    await this.renderAxis(ctx);
    await this.renderYAxes(ctx);
  }

  private async renderAxis(ctx: WXAxisContext): Promise<void> {}

  private async renderYAxes(ctx: WXAxisContext): Promise<void> {}
}
