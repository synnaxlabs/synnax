import { GLRenderContext } from "../gl/renderer";
import { WLinePlot } from "../LinePlot/LinePlot";
import { TelemProvider, TelemService } from "../telem";

export interface WComponent {
  key: string;
}

export interface WorkerMessage {
  root: string;
  key: string;
  type: string;
  props: any;
}

export class Worker {
  plots: WLinePlot[];
  telem: TelemService;

  constructor(telem: TelemService) {
    this.telem = telem;
    this.plot = [];
  }

  handle(msg: WorkerMessage): void {
    const plot = this.plots.find((p) => p.key === msg.root);
    if (plot != null) plot.handle(msg);
    else this.plots.push(new WLinePlot(msg, this.telem.provider()));
  }
}

const worker: Worker | null = null;

onmessage = (msg: MessageEvent<WorkerMessage>) => {
  if (msg.type === "worker") {
    const telem = new TelemService();
  }
};
