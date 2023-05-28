import { BoxT } from "@synnaxlabs/x";

import { GLLine, LineProgram, GLLineFactory } from "../Line/WLine";
import { TelemService } from "../telem";
import { WComponent, WorkerMessage } from "../worker/worker";

import { WYAxis } from "./YAxis";

import { WXAxis } from "@/core/vis/LinePlot/XAxis";

export interface WLinePlotProps {
  key: string;
  /** The region the plot is being renderered in. */
  region: BoxT;
  /** Viewport  represents the zoom and pan state of the plot. */
  viewport: BoxT;
}

export class LinePlot implements WComponent {
  props: WLinePlotProps;
  axes: WXAxis[];
  lineService: GLLineFactory;

  constructor(props: WLinePlotProps, lineService: GLLineFactory) {
    this.props = props;
    this.axes = [];
    this.lineService = lineService;
  }

  get key(): string {
    return this.props.key;
  }

  setProps(props: WLinePlotProps): void {
    this.props = props;
  }

  private async render(): Promise<void> {}

  handle(msg: WorkerMessage): void {
    switch (msg.type) {
      case GLLine.TYPE:
        return this.udpateLine(msg);
      case WXAxis.TYPE:
        return this.udpateXAxis(msg);
      case WYAxis.TYPE:
        return this.udpateYAxis(msg);
    }
    this._requestRender();
  }

  private _requestRender(): void {
    this.requestRender(async () => await this.render());
  }

  private udpateLine(msg: WorkerMessage): void {
    this.axes.forEach((a) =>
      a.axes.forEach((a) =>
        a.lines.forEach((l) => {
          if (l.key === msg.key) l.setProps(msg.props);
          else
            a.lines.push(
              new GLLine(
                msg.props,
                this.lineProgram,
                this._requestRender,
                this.telem.provider()
              )
            );
        })
      )
    );
  }

  private udpateXAxis(msg: WorkerMessage): void {
    this.axes.forEach((a) => {
      if (a.key === msg.key) a.setProps(msg.props);
      else this.axes.push(new WXAxis(msg.props));
    });
  }

  private udpateYAxis(msg: WorkerMessage): void {
    this.axes.forEach((xAxis) =>
      xAxis.axes.forEach((a) => {
        if (a.key === msg.key) a.setProps(msg.props);
        else xAxis.axes.push(new WYAxis(msg.props));
      })
    );
  }
}
