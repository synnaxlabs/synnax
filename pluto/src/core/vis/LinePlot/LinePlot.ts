import { Box, BoxT, OuterLocation, XY, clamp } from "@synnaxlabs/x";

import { RenderContext } from "../render";
import { RenderQueue } from "../render/RenderQueue";
import { WorkerUpdate } from "../worker/worker";

import { YAxis } from "./YAxis";

import { Line, LineFactory } from "@/core/vis/Line";
import { XAxis, XAxisContext } from "@/core/vis/LinePlot/XAxis";

export interface LinePlotProps {
  key: string;
  /** The region the plot is being renderered in. */
  region: BoxT;
  /** Viewport  represents the zoom and pan state of the plot. */
  viewport: BoxT;
  clearOverscan: number | XY;
}

const AXIS_WIDTH = 15;
const BASE_AXIS_PADDING = 12.5;

export class LinePlot {
  ctx: RenderContext;
  props: LinePlotProps;
  axes: XAxis[];
  lines: LineFactory;
  renderQueue: RenderQueue;

  constructor(
    ctx: RenderContext,
    props: LinePlotProps,
    lines: LineFactory,
    renderQueue: RenderQueue
  ) {
    this.ctx = ctx;
    this.props = props;
    this.axes = [];
    this.lines = lines;
    this.renderQueue = renderQueue;
  }

  get key(): string {
    return this.props.key;
  }

  private setProps(props: LinePlotProps): void {
    this.props = props;
    this._requestRender();
  }

  private async render(): Promise<void> {
    const region = new Box(this.props.region);
    const viewport = new Box(this.props.viewport);
    this.ctx.erase(
      region,
      typeof this.props.clearOverscan === "number"
        ? { x: this.props.clearOverscan, y: this.props.clearOverscan }
        : this.props.clearOverscan
    );
    const [axisPositions, axisSizes] = this.calculateAxisPositions(region);
    await Promise.all(
      this.axes.map(async (xAxis) => {
        const ctx: XAxisContext = {
          region,
          viewport,
          yAxisPositions: axisPositions[0][1],
          yAxisSizes: axisSizes[0][1],
          position: axisPositions[0][0],
          size: axisSizes[0][0],
        };
        await xAxis.render(ctx);
      })
    );
  }

  handle(update: WorkerUpdate): void {
    if (update.key === this.key) return this.setProps(update.props);
    switch (update.type) {
      case Line.TYPE:
        return this.udpateLine(update);
      case XAxis.TYPE:
        return this.updateXAxis(update);
      case YAxis.TYPE:
        return this.udpateYAxis(update);
    }
    this._requestRender();
  }

  private _requestRender(): void {
    this.renderQueue.push(async () => await this.render());
  }

  private udpateLine(u: WorkerUpdate): void {
    this.axes.forEach((a) =>
      a.axes.forEach((a) =>
        a.lines.forEach((l) => {
          if (l.key === u.key) l.setProps(u.props);
          else a.lines.push(this.lines.new(u.props, this._requestRender));
        })
      )
    );
  }

  private updateXAxis(u: WorkerUpdate): void {
    this.axes.forEach((a) => {
      if (a.key === u.key) a.setProps(u.props);
      else this.axes.push(new XAxis(this.ctx, u.props));
    });
  }

  private udpateYAxis(u: WorkerUpdate): void {
    this.axes.forEach((xAxis) =>
      xAxis.axes.forEach((a) => {
        if (a.key === u.key) a.setProps(u.props);
        else xAxis.axes.push(new YAxis(this.ctx, u.props));
      })
    );
  }

  private calculateAxisPositions(region: Box): [AxisPositions, AxisSizes] {
    const yAxes = this.axes.flatMap((xAxis) => xAxis.axes);
    const axisCounts: Record<OuterLocation, number> = {
      top: this.axes.length > 1 ? 1 : 0,
      bottom: this.axes.length > 0 ? 1 : 0,
      left: clamp(yAxes.length, 0, 2),
      right: clamp(yAxes.length - 2, 0, 2),
    };
    const axisPositions: AxisPositions = [];
    const axisSizes: AxisSizes = [];
    if (axisCounts.bottom > 0) {
      const x = BASE_AXIS_PADDING + axisCounts.left * AXIS_WIDTH;
      const y = region.height - BASE_AXIS_PADDING - AXIS_WIDTH;
      const size = region.width - x - BASE_AXIS_PADDING - axisCounts.right * AXIS_WIDTH;
      const axis = this.axes[0];
      const yAxes = axis.axes.map((a, i) => {
        const y = axisCounts.top * AXIS_WIDTH + BASE_AXIS_PADDING;
        const size = region.height - y - BASE_AXIS_PADDING - AXIS_WIDTH;
        if (i < 2) {
          const x = BASE_AXIS_PADDING + i * AXIS_WIDTH;
          return [{ x, y }, size];
        } else {
          const x = region.width - BASE_AXIS_PADDING - AXIS_WIDTH * (i - 2);
          return [{ x, y }, size];
        }
      });
      axisPositions.push([{ x, y }, yAxes.map(([pos]) => pos) as XY[]]);
      axisSizes.push([size, yAxes.map(([, size]) => size) as number[]]);
    }
    return [axisPositions, axisSizes];
  }
}

type AxisPositions = Array<[XY, XY[]]>;
type AxisSizes = Array<[number, number[]]>;
