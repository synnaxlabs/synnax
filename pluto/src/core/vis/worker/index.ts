import { Box, BoxT, TypedWorker } from "@synnaxlabs/x";

import { LineFactory } from "@/core/vis/Line/LineGL";
import { LinePlot, LinePlotUpdate } from "@/core/vis/LinePlot/LinePlot";
import { RenderContext } from "@/core/vis/render";
import { RenderQueue } from "@/core/vis/render/RenderQueue";
import { TelemProvider } from "@/core/vis/telem/TelemService";

export interface WorkerMessage {
  type: string;
  data: any;
}

interface BootstrapProps extends ResizeProps {
  canvas: OffscreenCanvas;
}

interface ResizeProps {
  dpr: number;
  box: BoxT;
}

export class VisWorker {
  worker: TypedWorker<WorkerMessage>;
  ctx: RenderContext | null = null;
  lines: LineFactory | null = null;
  queue: RenderQueue;
  plots: LinePlot[];
  telem: TelemProvider;

  constructor(telem: TelemProvider, worker: TypedWorker<WorkerMessage>) {
    this.worker = worker;
    this.telem = telem;
    this.plots = [];
    this.queue = new RenderQueue();

    worker.handle((msg) => this.handle(msg));
  }

  handle(msg: WorkerMessage): void {
    switch (msg.type) {
      case "bootstrap":
        this.bootstrap(msg.data);
        return;
      case "resize":
        this.resize(msg.data);
        return;
      case "set-props":
        this.updatePlot(msg.data);
    }
  }

  updatePlot(msg: LinePlotUpdate): void {
    // get first element in . separated path
    const first = msg.path.split(".")[0];
    const plot = this.plots.find((el) => el.key === first);
    if (plot == null) throw new Error(`Could not find plot with key ${first}`);
    plot.update(msg);
  }

  bootstrap(props: BootstrapProps): void {
    const ctx = new RenderContext(props.canvas, new Box(props.box), props.dpr);
    const lines = new LineFactory(ctx, this.telem);
    this.ctx = ctx;
    this.lines = lines;
  }

  resize(props: ResizeProps): void {
    if (this.ctx == null) throw new Error("Cannot resize before bootstrap");
    this.ctx.updateCanvasRegion(new Box(props.box), props.dpr);
    this.plots.forEach((el) => el.requestRender());
  }
}
