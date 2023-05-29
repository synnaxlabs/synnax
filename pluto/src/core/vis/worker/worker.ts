import { LineFactory } from "../Line";
import { LinePlot, LinePlotProps } from "../LinePlot/LinePlot";
import { RenderContext } from "../render";
import { RenderQueue } from "../render/RenderQueue";
import { TelemProvider } from "../telem/TelemService";

export interface WorkerUpdate {
  path: string;
  type: string;
  props: any;
}

export class Worker {
  ctx: RenderContext;
  queue: RenderQueue;
  plots: LinePlot[];
  telem: TelemProvider;
  lines: LineFactory;

  constructor(
    ctx: RenderContext,
    queue: RenderQueue,
    telem: TelemProvider,
    lines: LineFactory
  ) {
    this.telem = telem;
    this.plots = [];
    this.ctx = ctx;
    this.queue = queue;
    this.lines = lines;
  }

  handle(u: WorkerUpdate): void {
    const plot = this.plots.find((p) => p.key === u.root);
    if (plot != null) plot.handle(u);
    else
      this.plots.push(
        new LinePlot(this.ctx, u.props as LinePlotProps, this.lines, this.queue)
      );
  }
}
