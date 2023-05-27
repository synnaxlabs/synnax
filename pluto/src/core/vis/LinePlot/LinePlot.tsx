import { BoxT } from "@synnaxlabs/x";

import { WLine, WLineProgram } from "../Line/WLine";
import { TelemProvider } from "../telem";
import { WComponent, WorkerMessage } from "../worker/worker";

import { WYAxis } from "./YAxis";

import { WXAxis } from "@/core/vis/LinePlot/XAxis";

export interface WLinePlotProps {
  key: string;
  region: BoxT;
  viewport: BoxT;
}

export class WLinePlot implements WComponent {
  props: WLinePlotProps;
  axes: WXAxis[];
  key: string;
  telemProv: TelemProvider;
  lineProgram: WLineProgram;
  requestRender: (cbk: () => Promise<void>) => void;

  constructor(props: WLinePlotProps, telemProv: TelemProvider) {
    this.key = props.key;
    this.props = props;
    this.axes = [];
  }

  setProps(props: WLinePlotProps): void {
    this.props = props;
  }

  private async render(): Promise<void> {}

  handle(msg: WorkerMessage): void {
    switch (msg.type) {
      case WLine.TYPE:
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
          if (l.key === msg.key) {
            l.setProps(msg.props);
          } else {
            a.lines.push(
              new WLine(
                msg.props,
                this.lineProgram,
                this._requestRender,
                this.telemProv
              )
            );
          }
        })
      )
    );
  }

  private udpateXAxis(msg: WorkerMessage): void {
    this.axes.forEach((a) => {
      if (a.key === msg.key) {
        a.setProps(msg.props);
      } else {
        this.axes.push(new WXAxis(msg.props));
      }
    });
  }

  private udpateYAxis(msg: WorkerMessage): void {
    this.axes.forEach((xAxis) => {
      xAxis.axes.forEach((a) => {
        if (a.key === msg.key) {
          a.setProps(msg.props);
        } else {
          xAxis.axes.push(new WYAxis(msg.props));
        }
      });
    });
  }
}
