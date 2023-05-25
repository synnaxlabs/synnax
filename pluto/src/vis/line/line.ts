import { VisContext } from "@/vis/context";
import { Axes, AxesState } from "@/vis/line/axes";
import { Bounds, BoundsState } from "@/vis/line/bounds";
import { Channels, ChannelsState } from "@/vis/line/channels";
import { Lines, LinesState } from "@/vis/line/lines";
import { Ranges, RangesState } from "@/vis/line/ranges";
import { Scales } from "@/vis/line/scales";
import { Telem } from "@/vis/line/telem";
import { Viewport, ViewportState } from "@/vis/line/viewport";

export interface LineVisState {
  viewport: ViewportState;
  channels: ChannelsState;
  ranges: RangesState;
  styles: LinesState;
  axes: AxesState;
  bounds: BoundsState;
}

export class LineVis {
  viewport: Viewport;
  channels: Channels;
  ranges: Ranges;
  telem: Telem;
  bounds: Bounds;
  scales: Scales;
  axes: Axes;
  lines: Lines;

  constructor() {
    this.viewport = new Viewport();
    this.channels = new Channels();
    this.ranges = new Ranges();
    this.telem = new Telem();
    this.bounds = new Bounds();
    this.scales = new Scales();
    this.axes = new Axes();
    this.lines = new Lines();
  }

  static zeroState(): LineVisState {
    return {
      viewport: Viewport.zeroState(),
      channels: Channels.zeroState(),
      ranges: Ranges.zeroState(),
      axes: Axes.zeroState(),
      bounds: Bounds.zeroState(),
      styles: Lines.zeroState(),
    };
  }

  update(state: LineVisState): void {
    this.viewport.update(state.viewport);
    this.channels.update(state.channels);
    this.ranges.update(state.ranges);
    this.bounds.update(state.bounds);
    this.axes.update(state.axes);
    this.bounds.update(state.bounds);
  }

  async render(ctx: VisContext): Promise<void> {
    this.lines.render(ctx);
  }

  async build(ctx: VisContext): Promise<void> {
    await this.channels.build(ctx);
    await this.telem.build(ctx, this.channels, this.ranges);
    this.bounds.build(this.telem, 0);
    this.scales.build(this.viewport, this.bounds);
    this.axes.build(this.viewport, this.scales);
    this.lines.build(this.viewport, this.telem, this.scales, this.axes, ctx.theme);
  }
}
